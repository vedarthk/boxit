package cmd

import (
	"errors"
	"fmt"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
	"github.com/dustin/go-humanize"
	"github.com/mitchellh/ioprogress"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

const chunkSize int64 = 1 << 24

func uploadChunkedFile(dbx files.Client, r io.Reader, commitInfo *files.CommitInfo, sizeTotal int64) (err error) {
	res, err := dbx.UploadSessionStart(files.NewUploadSessionStartArg(),
		&io.LimitedReader{R: r, N: chunkSize})
	if err != nil {
		return
	}

	written := chunkSize

	for (sizeTotal - written) > chunkSize {
		args := files.NewUploadSessionCursor(res.SessionId, uint64(written))
		err = dbx.UploadSessionAppend(args, &io.LimitedReader{R: r, N: chunkSize})
		if err != nil {
			return
		}

		written += chunkSize
	}

	cursor := files.NewUploadSessionCursor(res.SessionId, uint64(written))
	args := files.NewUploadSessionFinishArg(cursor, commitInfo)

	if _, err = dbx.UploadSessionFinish(args, r); err != nil {
		return
	}

	return
}

func checkDirExists(dbx files.Client, dst string) (err error) {
	arg := files.NewListFolderArg(dst)

	_, err = dbx.ListFolder(arg)
	if err != nil {
		switch e := err.(type) {
		case files.ListFolderAPIError:
			if e.EndpointError.Path.Tag == files.LookupErrorNotFound {
				arg := files.NewCreateFolderArg(dst)
				if _, err = dbx.CreateFolder(arg); err != nil {
					return
				}
			}
		default:
			return err
		}
	}
	return
}

func fileExists(dbx files.Client, dst string) bool {
	arg := files.NewGetMetadataArg(dst)

	if _, err := dbx.GetMetadata(arg); err != nil {
		return false
	}
	return true
}

func uploadFile(src string, dst string) (err error) {
	contents, err := os.Open(src)
	defer contents.Close()
	if err != nil {
		return
	}

	contentsInfo, err := contents.Stat()
	if err != nil {
		return
	}

	progressbar := &ioprogress.Reader{
		Reader: contents,
		DrawFunc: ioprogress.DrawTerminalf(os.Stderr, func(progress, total int64) string {
			return fmt.Sprintf("Uploading %s %s/%s", path.Base(src), humanize.IBytes(uint64(progress)), humanize.IBytes(uint64(total)))
		}),
		Size: contentsInfo.Size(),
	}

	commitInfo := files.NewCommitInfo(dst)
	commitInfo.Mode.Tag = "overwrite"

	commitInfo.ClientModified = time.Now().UTC().Round(time.Second)

	dbx := files.New(config)

	err = checkDirExists(dbx, path.Dir(dst))
	if err != nil {
		return
	}

	if fileExists(dbx, dst) {
		fmt.Println("File exists!")
		return
	}

	if contentsInfo.Size() > chunkSize {
		return uploadChunkedFile(dbx, progressbar, commitInfo, contentsInfo.Size())
	}

	if _, err = dbx.Upload(commitInfo, progressbar); err != nil {
		return
	}

	return
}

func uploadDir(src string, dst string) (err error) {
	err = filepath.Walk(src, func(filepath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Println(dst + path.Base(filepath))
			return uploadFile(filepath, dst+path.Base(filepath))
		}
		return err
	})
	return
}

func upload(cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 || len(args) > 2 {
		return errors.New("`upload` requires `src` and/or `dst` arguments")
	}

	src := args[0]
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) {
		return
	}

	dst := "/"
	if len(args) == 2 {
		dst = args[1]
	}

	if srcInfo.IsDir() {
		return uploadDir(src, dst)
	} else {
		dst := dst + path.Base(src)
		return uploadFile(src, dst)
	}
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload files",
	RunE:  upload,
}

func init() {
	RootCmd.AddCommand(uploadCmd)
}
