// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitlab

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/drone/go-scm/scm"
)

type contentService struct {
	client *wrapper
}

func (s *contentService) Find(ctx context.Context, repo, path, ref string) (*scm.Content, *scm.Response, error) {
	path = url.QueryEscape(path)
	path = strings.Replace(path, ".", "%2E", -1)
	endpoint := fmt.Sprintf("api/v4/projects/%s/repository/files/%s?ref=%s", encode(repo), path, ref)
	out := new(content)
	res, err := s.client.do(ctx, "GET", endpoint, nil, out)
	raw, berr := base64.StdEncoding.DecodeString(out.Content)
	if berr != nil {
		// samples in the gitlab documentation use RawStdEncoding
		// so we fallback if StdEncoding returns an error.
		raw, berr = base64.RawStdEncoding.DecodeString(out.Content)
		if berr != nil {
			return nil, res, err
		}
	}
	return &scm.Content{
		Path: out.FilePath,
		Data: raw,
	}, res, err
}

func (s *contentService) Create(ctx context.Context, repo, path string, params *scm.ContentParams) (*scm.Response, error) {
	path = url.QueryEscape(path)
	path = strings.Replace(path, ".", "%2E", -1)
	endpoint := fmt.Sprintf("api/v4/projects/%s/repository/files/%s", encode(repo), path)
	in := new(createUpdateContent)
	in.Branch = params.Branch
	in.Content = params.Data
	in.CommitMessage = params.Message
	in.Encoding = "base64"
	in.AuthorEmail = params.Signature.Email
	in.AuthorName = params.Signature.Name
	res, err := s.client.do(ctx, "POST", endpoint, in, nil)
	return res, err

}

func (s *contentService) Update(ctx context.Context, repo, path string, params *scm.ContentParams) (*scm.Response, error) {
	path = url.QueryEscape(path)
	path = strings.Replace(path, ".", "%2E", -1)
	endpoint := fmt.Sprintf("api/v4/projects/%s/repository/files/%s", encode(repo), path)
	in := new(createUpdateContent)
	in.Branch = params.Branch
	in.Content = params.Data
	in.CommitMessage = params.Message
	in.Encoding = "base64"
	in.AuthorEmail = params.Signature.Email
	in.AuthorName = params.Signature.Name
	res, err := s.client.do(ctx, "PUT", endpoint, in, nil)
	return res, err
}

func (s *contentService) Delete(ctx context.Context, repo, path, ref string) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

func (s *contentService) List(ctx context.Context, repo, path, ref string, opts scm.ListOptions) ([]*scm.ContentInfo, *scm.Response, error) {
	endpoint := fmt.Sprintf("api/v4/projects/%s/repository/tree?path=%s&ref=%s&%s", encode(repo), url.QueryEscape(path), ref, encodeListOptions(opts))
	out := []*object{}
	res, err := s.client.do(ctx, "GET", endpoint, nil, &out)
	return convertContentInfoList(out), res, err
}

type content struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	Size         int    `json:"size"`
	Encoding     string `json:"encoding"`
	Content      string `json:"content"`
	Ref          string `json:"ref"`
	BlobID       string `json:"blob_id"`
	CommitID     string `json:"commit_id"`
	LastCommitID string `json:"last_commit_id"`
}

type createUpdateContent struct {
	Branch        string `json:"branch"`
	Content       []byte `json:"content"`
	CommitMessage string `json:"commit_message"`
	Encoding      string `json:"encoding"`
	AuthorEmail   string `json:"author_email"`
	AuthorName    string `json:"author_name"`
}

type object struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

func convertContentInfoList(from []*object) []*scm.ContentInfo {
	to := []*scm.ContentInfo{}
	for _, v := range from {
		to = append(to, convertContentInfo(v))
	}
	return to
}

func convertContentInfo(from *object) *scm.ContentInfo {
	to := &scm.ContentInfo{Path: from.Path}
	// See the following link for supported file modes:
	// https://godoc.org/gopkg.in/src-d/go-git.v4/plumbing/filemode
	switch mode, _ := strconv.ParseInt(from.Mode, 8, 32); mode {
	case 0100644, 0100664, 0100755:
		to.Kind = scm.ContentKindFile
	case 0040000:
		to.Kind = scm.ContentKindDirectory
	case 0120000:
		to.Kind = scm.ContentKindSymlink
	case 0160000:
		to.Kind = scm.ContentKindGitlink
	default:
		to.Kind = scm.ContentKindUnsupported
	}
	return to
}
