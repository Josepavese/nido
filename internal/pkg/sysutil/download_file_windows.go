package sysutil

import (
	"golang.org/x/sys/windows"
	"os"
)

func downloadFileLinks(file *os.File) (uint64, error) {
	var info windows.ByHandleFileInformation
	err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info)
	return uint64(info.NumberOfLinks), err
}
