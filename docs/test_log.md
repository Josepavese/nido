# VM Ops Test Log

Environment:
- Mocked libvirt/qemu tools
- Config: /tmp/tmp.OXObYRk6Bt/config.env

Commands executed:
- nido -h
- nido ls (empty)
- nido create vm-test-1
- nido info vm-test-1
- nido stop vm-test-1
- nido start vm-test-1
- nido ls
- nido delete vm-test-1
- nido prune
- nido template base-vm template-headless-2

Outputs (captured):
- help: /tmp/vmops_test_help.txt
- info: /tmp/vmops_test_info.txt
- start: /tmp/vmops_test_start.txt
- list: /tmp/vmops_test_list.txt
- cleanup: /tmp/vmops_test_cleanup.txt
- template: /tmp/vmops_test_template.txt
