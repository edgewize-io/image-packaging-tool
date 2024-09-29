package lock

import (
	"fmt"
	"os"
)

func LockFile(lockFilePath string) bool {
	lockFile, err := os.Create(lockFilePath)
	if err != nil {
		fmt.Printf("Error create lock file: %v", err)
		return false
	}

	defer lockFile.Close()

	return true
}

func UnlockFile(lockFilePath string) {
	if err := os.Remove(lockFilePath); err != nil {
		fmt.Println("Error unlocking file:", err)
	}
}
