package extension

import (
	"encoding/json"
	"os"
)

type fileMetadata struct {
	Owner        extensionName `json:"owner"`
	ConfigMapKey string        `json:"configMapKey"`
}

type fileTracker struct {
	Files map[string]fileMetadata `json:"files"`
}

func (c *extensionContext) loadFileTracker() (*fileTracker, error) {
	var tracker fileTracker
	data, err := os.ReadFile(c.fileTrackerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &fileTracker{
				Files: make(map[string]fileMetadata),
			}, nil
		}
		return nil, err
	}
	err = json.Unmarshal(data, &tracker)
	if err != nil {
		return nil, err
	}
	return &tracker, nil
}

func (c *extensionContext) saveFileTracker(tracker *fileTracker) error {
	bytes, err := json.Marshal(*tracker)
	if err != nil {
		return err
	}
	err = os.WriteFile(c.fileTrackerPath, bytes, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (t *fileTracker) getFilesByOwner(owner extensionName) []string {
	files := make([]string, 0)
	for file, meta := range t.Files {
		if meta.Owner == owner {
			files = append(files, file)
		}
	}
	return files
}

func (t *fileTracker) setMetadata(file string, meta fileMetadata) {
	t.Files[file] = meta
}

func (t *fileTracker) clearMetadata(file string) {
	delete(t.Files, file)
}

func (t *fileTracker) isTracked(file string) bool {
	_, exists := t.Files[file]
	return exists
}

func (t *fileTracker) isOwner(file string, name extensionName) bool {
	meta, exists := t.Files[file]
	if !exists {
		return false
	}
	return meta.Owner == name
}

func (t *fileTracker) getOwner(file string) extensionName {
	if meta, exists := t.Files[file]; exists {
		return meta.Owner
	} else {
		return ""
	}
}
