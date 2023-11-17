package extension

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func CreateFileTrackerMock() *fileTracker {
	return &fileTracker{
		Files: map[string]fileMetadata{
			"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua": fileMetadata{
				Owner:        "test-extension",
				ConfigMapKey: "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
		},
	}
}

func CreateFileMetaData() fileMetadata {
	return fileMetadata{
		Owner:        "test-extension",
		ConfigMapKey: "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua",
	}
}

func TestSaveFileTraker(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	err := extensionContext.saveFileTracker(CreateFileTrackerMock())
	assert.Nil(t, err)
}

func TestLoadFileTraker(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	actualFileTracker, err := extensionContext.loadFileTracker()
	assert.Nil(t, err)
	assert.Equal(t, &fileTracker{Files: map[string]fileMetadata{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua": fileMetadata{Owner: "test-extension", ConfigMapKey: "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"}}}, actualFileTracker)
}

func TestGetFilesByOwner(t *testing.T) {
	fileTracker := CreateFileTrackerMock()
	list := fileTracker.getFilesByOwner("test-extension")
	assert.Equal(t, []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"}, list)
}

func TestSetMetadata(t *testing.T) {
	fileTracker := CreateFileTrackerMock()
	file := "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"
	metaData := CreateFileMetaData()
	fileTracker.setMetadata(file, metaData)
	assert.Equal(t, fileTracker.Files[file], metaData)
}

func TestClearMetadata(t *testing.T) {
	fileTracker := CreateFileTrackerMock()
	file := "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"
	fileTracker.clearMetadata(file)
	assert.Equal(t, fileMetadata(fileMetadata{Owner: "", ConfigMapKey: ""}), fileTracker.Files[file])
}

func TestIsTracked(t *testing.T) {
	fileTracker := CreateFileTrackerMock()
	file := "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"
	isTracked := fileTracker.isTracked(file)
	assert.Equal(t, bool(true), isTracked)
}

func TestIsOwner(t *testing.T) {
	fileTracker := CreateFileTrackerMock()
	file := "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"
	isOwner := fileTracker.isOwner(file, "test-extension")
	assert.Equal(t, bool(true), isOwner)
}

func TestGetOwner(t *testing.T) {
	fileTracker := CreateFileTrackerMock()
	file := "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"
	owner := fileTracker.getOwner(file)
	assert.Equal(t, extensionName("test-extension"), owner)
}
