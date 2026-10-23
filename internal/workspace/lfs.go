// This file is part of PeerFoil.
// internal/workspace/lfs.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Detects unsupported LFS pointers in actual Git blobs, including pointers
// created by repository-private or global attributes absent from the tracked tree.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// The LFS specification limits pointers to fewer than 1,024 bytes. Batch inspection
// avoids one process per candidate and recognizes both current and legacy versions.
// See https://github.com/git-lfs/git-lfs/blob/main/docs/spec.md.
func (r *Repo) rejectLFSPointers(ctx context.Context, objects []string) error {
	if len(objects) == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "cat-file", "--batch")
	cmd.Dir = r.Root
	cmd.Stdin = strings.NewReader(strings.Join(objects, "\n") + "\n")
	output, err := cmd.Output()
	if err != nil {
		return errs.New(errs.Git, "could not inspect source blobs for LFS pointers", "check Git availability before retrying")
	}
	reader := bufio.NewReader(bytes.NewReader(output))
	for _, expected := range objects {
		header, err := reader.ReadString('\n')
		var object, kind string
		var size int
		if err != nil {
			return errs.New(errs.Git, "source blob inspection returned incomplete output", "")
		}
		if _, err := fmt.Sscanf(header, "%s %s %d", &object, &kind, &size); err != nil || object != expected || kind != "blob" || size < 0 || size >= 1024 {
			return errs.New(errs.Git, "source blob inspection returned an invalid object", "")
		}
		data := make([]byte, size+1)
		if _, err := io.ReadFull(reader, data); err != nil || data[size] != '\n' {
			return errs.New(errs.Git, "source blob inspection returned incomplete content", "")
		}
		for _, prefix := range []string{"version https://git-lfs.github.com/spec/v1\n", "version https://hawser.github.com/spec/v1\n"} {
			if bytes.HasPrefix(data[:size], []byte(prefix)) {
				return errs.New(errs.Policy, "the source tree contains a Git LFS pointer, which this release does not support",
					"replace LFS pointers with regular content through change intake before continuing")
			}
		}
	}
	return nil
}
