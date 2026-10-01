//go:build windows

package connections

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func buildPIDNameMap() (map[int]string, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return map[int]string{}, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	result := make(map[int]string)
	err = windows.Process32First(snapshot, &entry)
	for err == nil {
		pid := int(entry.ProcessID)
		name := windows.UTF16ToString(entry.ExeFile[:])
		result[pid] = name
		err = windows.Process32Next(snapshot, &entry)
	}

	return result, nil
}
