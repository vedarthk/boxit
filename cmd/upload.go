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
			return fmt.Sprintf("Uploading %s/%s", humanize.IBytes(uint64(progress)), humanize.IBytes(uint64(total)))
		}),
		Size: contentsInfo.Size(),
	}

	commitInfo := files.NewCommitInfo(dst)
	commitInfo.Mode.Tag = "overwrite"

	commitInfo.ClientModified = time.Now().UTC().Round(time.Second)

	dbx := files.New(config)

	// if contentsInfo.Size() > chunkSize {
	// 	return uploadChunkedFile()
	// }

	if _, err = dbx.Upload(commitInfo, progressbar); err != nil {
		return
	}

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

	if srcInfo.IsDir() {
		fmt.Print(path.Base(src))
	} else {
		dst := "/" + path.Base(src)
		return uploadFile(src, dst)
	}

	return
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload files",
	RunE:  upload,
}

func init() {
	RootCmd.AddCommand(uploadCmd)
}