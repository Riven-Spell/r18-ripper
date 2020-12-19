package playlist

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"
)

type Handler struct {
	URI           url.URL
	TargetBitrate int64
	HLSPlaylist   *HLSPlaylist
}

type ChunkListReturn struct {
	Key HLSKey
	GlobalHeaders map[string]string
	LocalHeaders map[string]string
	Url string
	ChunkID int
}

func (h *Handler) GetChunks() (chan ChunkListReturn, int, error) {
	baseURI := h.URI
	baseURI.Path = path.Dir(baseURI.Path)

	resp, err := http.Get(h.URI.String())

	if err != nil {
		return nil, 0, err
	}

	buf, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, 0, err
	}

	playlist := ParseHLSPlaylist(string(buf))

	// jank way to detect playlist
	if strings.Contains(h.URI.Path, "playlist.m3u8") {
		var entry HLSPlaylistEntry
		maxBitRate := int64(0)
		for _,v := range playlist.Entries {
			var bitrate int64
			fmt.Sscanf(v.Headers["EXT-X-STREAM-INF"], "BANDWIDTH=%d", &bitrate)

			if h.TargetBitrate != 0 {
				if bitrate == h.TargetBitrate {
					entry = v
					break
				}
			} else if bitrate > maxBitRate {
				maxBitRate = bitrate
				entry = v
			}
		}

		target := strings.Split(entry.Target, "?")

		tmpURI := baseURI
		tmpURI.Path = path.Join(tmpURI.Path, target[0])

		if len(target) > 1 {
			tmpURI.RawQuery = target[1]
		}

		resp, err := http.Get(tmpURI.String())

		if err != nil {
			return nil, 0, err
		}

		buf, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, 0, err
		}

		playlist = ParseHLSPlaylist(string(buf))
	}

	h.HLSPlaylist = &playlist
	hlsKey, err := playlist.GetKey()

	if err != nil {
		return nil, 0, err
	}

	baseURI.RawQuery = ""

	outCh := make(chan ChunkListReturn, runtime.NumCPU())
	go func() {
		chunkID := 0 // TODO: add sequence number, but for now, r18 always uses 0.
		for _, v := range playlist.Entries {
			tmpURI := baseURI

			tmpURI.Path = path.Join(tmpURI.Path, v.Target)

			outCh <- ChunkListReturn{
				Key:           hlsKey,
				GlobalHeaders: playlist.Headers,
				LocalHeaders:  v.Headers,
				Url:           tmpURI.String(),
				ChunkID: 	   chunkID,
			}

			chunkID++
		}

		close(outCh)
	}()

	return outCh, len(playlist.Entries), err
}