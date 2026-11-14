//go:build windows

package core

import "golang.org/x/sys/windows"

func systemDocuments(_ string) string {
	path, err := windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return ""
	}
	return path
}
