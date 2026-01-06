package provider

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CloudInit struct {
	Hostname       string
	User           string
	SSHKey         string
	CustomUserData string
}

func (c *CloudInit) GenerateISO(outPath string) error {
	tmpDir, err := os.MkdirTemp("", "nido-cloud-init")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 1. meta-data (One-line JSON for best compatibility with some minimal CirrOS parsers)
	metaData := fmt.Sprintf("{\"instance-id\": \"i-%s\", \"local-hostname\": \"%s\"}\n", c.Hostname, c.Hostname)
	if err := os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0644); err != nil {
		return err
	}

	// 2. user-data
	var userData string
	if c.CustomUserData != "" {
		userData = c.CustomUserData
	} else {
		userData = "#cloud-config\n"
		userData += "users:\n"
		userData += fmt.Sprintf("  - name: %s\n", c.User)
		userData += "    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n"
		if c.SSHKey != "" {
			userData += "    ssh_authorized_keys:\n"
			userData += fmt.Sprintf("      - %s\n", c.SSHKey)
		}
		userData += "ssh_pwauth: true\n"
		userData += "chpasswd:\n"
		userData += "  list: |\n"
		userData += fmt.Sprintf("    %s:nido\n", c.User)
		userData += "  expire: false\n"
		userData += "\n"
		userData += "runcmd:\n"
		userData += "  - if [ -f /etc/default/grub ]; then sed -i 's/GRUB_TIMEOUT=[0-9]*/GRUB_TIMEOUT=0/' /etc/default/grub && (update-grub || grub-mkconfig -o /boot/grub/grub.cfg); fi\n"
		userData += "  - if [ -f /boot/extlinux.conf ]; then sed -i 's/^TIMEOUT [0-9]*/TIMEOUT 0/' /boot/extlinux.conf; fi\n"
	}

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

func GetLocalSSHKey() string {
	// Try typical locations
	home, _ := os.UserHomeDir()
	files := []string{"id_ed25519.pub", "id_rsa.pub"}
	for _, f := range files {
		path := filepath.Join(home, ".ssh", f)
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}
