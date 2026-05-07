package provider

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/internal/pkg/sysutil"
)

// CloudInit handles the generation of cloud-init seed ISOs.
type CloudInit struct {
	Hostname       string
	User           string
	SSHKey         string
	CustomUserData string
}

// GenerateISO creates a cloud-init seed ISO using NoCloud format.
func (c *CloudInit) GenerateISO(outPath string) error {
	tmpDir, err := os.MkdirTemp("", "nido-cloud-init")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 1. meta-data
	metaData := fmt.Sprintf("{\"instance-id\": \"i-%s\", \"local-hostname\": \"%s\"}", c.Hostname, c.Hostname)
	if err := os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0644); err != nil {
		return err
	}

	// 2. user-data
	userData := c.buildUserData()
	if err := os.WriteFile(filepath.Join(tmpDir, "user-data"), []byte(userData), 0644); err != nil {
		return err
	}

	// 3. Create ISO/Disk
	if _, err := exec.LookPath("cloud-localds"); err == nil {
		// cloud-localds <output> <user-data> [meta-data]
		cmd := exec.Command("cloud-localds", outPath, filepath.Join(tmpDir, "user-data"), filepath.Join(tmpDir, "meta-data"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("cloud-localds failed: %v (%s)", err, string(output))
		}
		return nil
	}

	// Fallback to ISO generation (genisoimage/mkisofs)
	tool := "genisoimage"
	if _, err := exec.LookPath("genisoimage"); err != nil {
		if _, err := exec.LookPath("mkisofs"); err == nil {
			tool = "mkisofs"
		} else if _, err := exec.LookPath("xorriso"); err == nil {
			tool = "xorriso"
		} else {
			return fmt.Errorf("no suitable ISO creation tool found (install cloud-utils or genisoimage)")
		}
	}

	// genisoimage -output <iso> -volid cidata -joliet -rock <dir>
	cmd := exec.Command(tool, "-output", outPath, "-volid", "cidata", "-joliet", "-rock", tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iso creation failed: %v (%s)", err, string(output))
	}

	return nil
}

func (c *CloudInit) buildUserData() string {
	if c.User == "cirros" {
		return c.buildCirrOSUserData()
	}

	base := c.buildCloudConfigUserData()
	if strings.TrimSpace(c.CustomUserData) == "" {
		return base
	}
	return buildMultipartUserData(base, c.CustomUserData)
}

func (c *CloudInit) buildCirrOSUserData() string {
	// CirrOS's minimal cloud-init only supports shell scripts (starting with #!).
	userData := "#!/bin/sh\n"
	userData += "echo \"[Nido] cloud-init script starting...\" > /dev/console\n"
	if c.SSHKey != "" {
		userData += fmt.Sprintf("mkdir -p /home/%s/.ssh\n", c.User)
		userData += fmt.Sprintf("cat <<EOF >> /home/%s/.ssh/authorized_keys\n%s\nEOF\n", c.User, c.SSHKey)
		userData += fmt.Sprintf("chown -R %s:%s /home/%s/.ssh\n", c.User, c.User, c.User)
		userData += fmt.Sprintf("chmod 700 /home/%s/.ssh\n", c.User)
		userData += fmt.Sprintf("chmod 600 /home/%s/.ssh/authorized_keys\n", c.User)
		userData += "echo \"[Nido] SSH key injected.\" > /dev/console\n"
	}

	custom := strings.TrimSpace(c.CustomUserData)
	if custom == "" {
		return userData
	}
	if strings.HasPrefix(custom, "#!") {
		if idx := strings.Index(c.CustomUserData, "\n"); idx >= 0 {
			userData += "\n# Nido custom user-data\n"
			userData += c.CustomUserData[idx+1:]
			if !strings.HasSuffix(userData, "\n") {
				userData += "\n"
			}
		}
	} else {
		userData += "echo \"[Nido] Ignored non-shell custom user-data for CirrOS.\" > /dev/console\n"
	}
	return userData
}

func (c *CloudInit) buildCloudConfigUserData() string {
	userData := "#cloud-config\n"
	userData += "users:\n"
	userData += fmt.Sprintf("  - name: %s\n", c.User)
	userData += "    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n"
	if c.SSHKey != "" {
		userData += "    ssh_authorized_keys:\n"
		userData += fmt.Sprintf("      - %s\n", c.SSHKey)
	}
	if c.SSHKey != "" {
		userData += "ssh_pwauth: false\n"
	} else {
		userData += "ssh_pwauth: true\n"
		userData += "chpasswd:\n"
		userData += "  list: |\n"
		userData += fmt.Sprintf("    %s:nido\n", c.User)
		userData += "  expire: false\n"
	}
	userData += "\n"
	userData += "runcmd:\n"
	userData += "  - if [ -f /etc/default/grub ]; then sed -i 's/GRUB_TIMEOUT=[0-9]*/GRUB_TIMEOUT=0/' /etc/default/grub && (update-grub || grub-mkconfig -o /boot/grub/grub.cfg); fi\n"
	userData += "  - if [ -f /boot/extlinux.conf ]; then sed -i 's/^TIMEOUT [0-9]*/TIMEOUT 0/' /boot/extlinux.conf; fi\n"
	userData += fmt.Sprintf("  - if [ -x /usr/bin/doas ]; then mkdir -p /etc/doas.d && echo \"permit nopass %s as root\" > /etc/doas.d/nido.conf && chmod 0400 /etc/doas.d/nido.conf; fi\n", c.User)
	return userData
}

func buildMultipartUserData(baseCloudConfig, custom string) string {
	const boundary = "===============NIDO_USER_DATA_BOUNDARY=="

	contentType := "text/cloud-config"
	trimmed := strings.TrimSpace(custom)
	if strings.HasPrefix(trimmed, "#!") {
		contentType = "text/x-shellscript"
	}

	var b strings.Builder
	b.WriteString("MIME-Version: 1.0\n")
	b.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\n\n", boundary))
	b.WriteString(fmt.Sprintf("--%s\n", boundary))
	b.WriteString("Content-Type: text/cloud-config; charset=\"us-ascii\"\n\n")
	b.WriteString(baseCloudConfig)
	if !strings.HasSuffix(baseCloudConfig, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\n--%s\n", boundary))
	b.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"us-ascii\"\n\n", contentType))
	b.WriteString(custom)
	if !strings.HasSuffix(custom, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("--%s--\n", boundary))
	return b.String()
}

func GetLocalSSHKey() string {
	// Try typical locations
	home, _ := sysutil.UserHome()
	files := []string{"id_ed25519.pub", "id_rsa.pub"}
	for _, f := range files {
		path := filepath.Join(home, ".ssh", f)
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	nidoDir := filepath.Join(home, ".nido")
	keyPath := filepath.Join(nidoDir, "nido_ed25519")
	pubPath := keyPath + ".pub"
	if data, err := os.ReadFile(pubPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	if _, err := exec.LookPath("ssh-keygen"); err == nil {
		_ = os.MkdirAll(nidoDir, 0700)
		_ = os.Chmod(nidoDir, 0700)
		cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-C", "nido-local-vm-manager")
		if out, err := cmd.CombinedOutput(); err == nil {
			_ = out
			if data, readErr := os.ReadFile(pubPath); readErr == nil {
				return strings.TrimSpace(string(data))
			}
		}
	}
	return ""
}

func localNidoSSHKeyPath() string {
	home, err := sysutil.UserHome()
	if err != nil {
		return ""
	}
	keyPath := filepath.Join(home, ".nido", "nido_ed25519")
	if _, err := os.Stat(keyPath); err == nil {
		return keyPath
	}
	return ""
}
