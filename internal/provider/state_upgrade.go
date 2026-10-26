package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

const stateSchemaFile = ".state-schema"
const currentStateSchema = "architecture-v1\n"
const stateSchemaMode = 0600
const stateHomeMode = 0755

// PrepareState is the bounded installation bootstrap, not a legacy runtime
// decoder. The completion marker is published only after the resumable metadata
// conversion succeeds. Ordinary starts then use only the current schema.
func (p *QemuProvider) PrepareState(ctx context.Context) (retErr error) {
	marker := filepath.Join(p.RootDir, stateSchemaFile)
	complete, err := statePrepared(marker)
	if err != nil || complete {
		return err
	}
	if err := os.MkdirAll(p.RootDir, stateHomeMode); err != nil {
		return err
	}
	lease, err := sysutil.AcquireResourceContext(ctx, marker+".lock")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lease.Close()) }()
	complete, err = statePrepared(marker)
	if err != nil || complete {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := p.migrateArchitecture(false); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return sysutil.WriteFile(marker, []byte(currentStateSchema), stateSchemaMode)
}

func statePrepared(marker string) (bool, error) {
	info, err := os.Lstat(marker)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("state schema marker must be a regular file: %s", marker)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		return false, err
	}
	if string(data) != currentStateSchema {
		return false, fmt.Errorf("unsupported state schema in %s; existing data preserved", marker)
	}
	return true, nil
}
