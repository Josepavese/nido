# VM Ops Real Selftest Log

Config:
- POOL_PATH=/tmp/tmp.SIEMAIzAZF/pool
- BACKUP_DIR=/tmp/tmp.SIEMAIzAZF/backups
- VMS_POOL=vmops-selftest-649166
- TEMPLATE_DEFAULT=template-headless
- NETWORK_HOSTONLY=hostonly56
- NETWORK_NAT=default
- SELFTEST_PREFIX=vm-selftest

Commands executed:
- create vm-selftest-1
- info vm-selftest-1
VM: vm-selftest-1
Hostname: (unknown)
IP: (no-ip)
SSH: ssh vmuser@<ip>
- stop vm-selftest-1
Domain 'vm-selftest-1' is being shutdown

- start vm-selftest-1
VM started, but IP not found within 15s.
WARN: start did not resolve IP within timeout; VM may still be running.
- list-test
No test VMs found (NAME_REGEX=(^|[-_])test).
- destroy vm-selftest-1
Domain 'vm-selftest-1' destroyed

Domain 'vm-selftest-1' has been undefined

Vol vm-selftest-1.qcow2 deleted

VM destroyed: vm-selftest-1
- prune
No volumes in pool: vmops-selftest-649166
- create vm-selftest-base
Domain 'vm-selftest-base' is being shutdown

WARN: base VM did not shut off; forcing destroy.
- template vm-selftest-base vm-selftest-template

Compressed template created: /tmp/tmp.SIEMAIzAZF/backups/vm-selftest-template.compact.qcow2
- destroy vm-selftest-base
Domain 'vm-selftest-base' has been undefined

Vol vm-selftest-base.qcow2 deleted

VM destroyed: vm-selftest-base
- delete temp backup /tmp/tmp.SIEMAIzAZF/backups/vm-selftest-template.compact.qcow2
