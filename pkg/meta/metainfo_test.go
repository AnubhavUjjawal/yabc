package meta_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
)

func TestTorrentFilesCanBeLoadedIntoMetaInfo(t *testing.T) {
	bencoder := bencoding.NewBencoder()
	rootDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting root dir: %v", err)
	}
	files, err := ioutil.ReadDir(filepath.Join(rootDir, "..", "..", "sample_torrents"))
	if err != nil {
		t.Errorf("Error reading sample files: %v", err)
	}
	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(rootDir, "..", "..", "sample_torrents", file.Name()))
			if err != nil {
				t.Errorf("Error opening file: %v", err)
			}
			var metaInfo meta.MetaInfo
			err = bencoder.Unmarshal(string(content), &metaInfo)
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			t.Log("Read torrent file", metaInfo.Info.Name)
		})
	}

}
