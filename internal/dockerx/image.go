package dockerx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/moby/moby/client"
)

type ImageSummary struct {
	ID          string            `json:"id"`
	RepoTags    []string          `json:"repo_tags,omitempty"`
	RepoDigests []string          `json:"repo_digests,omitempty"`
	Created     time.Time         `json:"created"`
	Size        int64             `json:"size"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ImageRow is the primitive-only projection used by docker_image_list so the
// response stays in TOON's compact tabular form. Drops repo_digests + labels;
// shortens the id; emits the first repo tag.
type ImageRow struct {
	ID      string    `json:"id"` // shortened to 19 chars (sha256:<12>)
	Tag     string    `json:"tag"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
}

func (i ImageSummary) Row() ImageRow {
	id := i.ID
	// Image IDs are commonly "sha256:<64 hex>". Keep the algorithm prefix and a
	// short hash so the row is still human-recognizable.
	if len(id) > 19 {
		id = id[:19]
	}
	tag := ""
	if len(i.RepoTags) > 0 {
		tag = i.RepoTags[0]
	}
	return ImageRow{ID: id, Tag: tag, Size: i.Size, Created: i.Created}
}

func (h *Host) ListImages(ctx context.Context, all bool) ([]ImageSummary, error) {
	c, err := h.client()
	if err != nil {
		return nil, err
	}
	res, err := c.ImageList(ctx, client.ImageListOptions{All: all})
	if err != nil {
		return nil, err
	}
	out := make([]ImageSummary, 0, len(res.Items))
	for _, im := range res.Items {
		out = append(out, ImageSummary{
			ID:          im.ID,
			RepoTags:    im.RepoTags,
			RepoDigests: im.RepoDigests,
			Created:     time.Unix(im.Created, 0),
			Size:        im.Size,
			Labels:      im.Labels,
		})
	}
	return out, nil
}

// PullImage pulls an image reference and returns the daemon's progress
// stream as a single string of newline-delimited JSON. Callers can show that
// to a user or just ignore it; the call doesn't return until the pull is done.
func (h *Host) PullImage(ctx context.Context, ref string) (string, error) {
	c, err := h.client()
	if err != nil {
		return "", err
	}
	res, err := c.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer res.Close()
	// Drain to completion. We collapse to a status summary so the MCP response
	// is compact instead of a megabyte of layer progress.
	var lastStatus, lastError string
	dec := json.NewDecoder(res)
	for {
		var msg struct {
			Status      string `json:"status"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			return lastStatus, fmt.Errorf("decode pull progress: %w", err)
		}
		if msg.Error != "" {
			lastError = msg.Error
		} else if msg.ErrorDetail.Message != "" {
			lastError = msg.ErrorDetail.Message
		}
		if msg.Status != "" {
			lastStatus = msg.Status
		}
	}
	if lastError != "" {
		return lastStatus, fmt.Errorf("%s", lastError)
	}
	return lastStatus, nil
}

func (h *Host) RemoveImage(ctx context.Context, ref string, force, pruneChildren bool) error {
	c, err := h.client()
	if err != nil {
		return err
	}
	_, err = c.ImageRemove(ctx, ref, client.ImageRemoveOptions{Force: force, PruneChildren: pruneChildren})
	return err
}
