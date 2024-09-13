package utils

import "os"

func FileExistsAndReadable(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}
