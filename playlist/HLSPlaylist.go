package playlist

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

type HLSPlaylist struct {
	Headers map[string]string
	Entries []HLSPlaylistEntry
}

type HLSPlaylistEntry struct {
	Headers map[string]string
	Target string
}

type HLSKey struct {
	Method  string
	Key     []byte
	IV      []byte
	IVEmpty bool
}

func (h *HLSPlaylist) WritePlaylistToFile(fileName string) error {
	folder := path.Dir(fileName)
	key, err := h.GetKey()

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(folder, "r18key"), key.Key, 0644)

	if err != nil {
		return err
	}

	output := "#EXTM3U\n"

	for k,v := range h.Headers {
		output += "#" + k + ":"

		if k == "EXT-X-KEY" {
			entries := strings.Split(v, ",")
			for _,v := range entries {
				if strings.HasPrefix(v, "URI=") {
					output += "URI=\"r18key\""
				} else {
					output += v + ","
				}
			}

			output += "\n"
		} else {
			output += v + "\n"
		}
	}

	for _,v := range h.Entries {
		for k,v := range v.Headers {
			output += "#" + k + ":" + v + "\n"
		}

		output += v.Target + "\n"
	}

	output += "#EXT-X-ENDLIST\n"

	err = ioutil.WriteFile(fileName, []byte(output), 0644)

	if err != nil {
		return err
	}

	return err
}

func (h *HLSPlaylist) GetKey() (hlsKey HLSKey, err error) {
	segments := make([]string, 0)
	hlsKey = HLSKey{ IVEmpty: true }

	if _, ok := h.Headers["EXT-X-KEY"]; !ok {
		return HLSKey{ Method: "NONE", IVEmpty: true }, nil
	}

	cSeg := ""
	inStr := false
	for _,v := range h.Headers["EXT-X-KEY"] {
		switch {
		case inStr:
			if v == '"' {
				inStr = false
			}

			cSeg += string(v)
		case v == '"':
			inStr = true
			fallthrough
		case v != ',':
			cSeg += string(v)
		case v == ',':
			segments = append(segments, cSeg)
			cSeg = ""
		}
	}

	if cSeg != "" {
		segments = append(segments, cSeg)
	}

	for _,v := range segments {
		name := v[:strings.Index(v, "=")]
		value := v[strings.Index(v, "=") + 1:]

		switch name {
		case "METHOD":
			hlsKey.Method = value
		case "URI":
			var resp *http.Response
			resp, err = http.Get(strings.TrimSuffix(strings.TrimPrefix(value, "\""), "\""))

			if err != nil {
				return
			}

			hlsKey.Key, err = ioutil.ReadAll(resp.Body)

			if err != nil {
				return
			}
		case "IV":
			hlsKey.IV = make([]byte, 16)

			_, err = fmt.Sscanf(strings.TrimPrefix(value, "0x"), "%X", &hlsKey.IV)

			if err != nil {
				return
			}

			hlsKey.IVEmpty = false
		}
	}

	return hlsKey, nil
}

func ParseHLSPlaylist(s string) HLSPlaylist {
	out := HLSPlaylist{ Headers: map[string]string{}, Entries: make([]HLSPlaylistEntry, 0) }

	lines := strings.Split(s, "\n")
	cEntry := HLSPlaylistEntry{ Headers: map[string]string{} }

	for _,v := range lines {
		line := strings.TrimSuffix(v, "\r")

		if strings.HasPrefix(line, "#") {
			headerBits := strings.Split(line[1:], ":")

			if headerBits[0] == "EXTM3U" || headerBits[0] == "EXT-X-ENDLIST" {
				continue
			} else if strings.HasPrefix(headerBits[0], "EXT-X-") && headerBits[0] != "EXT-X-STREAM-INF" {
				// technically, these headers apply to sections of files...
				// However, R18 doesn't do that. So, instead of staying in-spec, I'm going to go out of spec with that assumption.
				out.Headers[headerBits[0]] = strings.Join(headerBits[1:], ":")
			} else {
				cEntry.Headers[headerBits[0]] = strings.Join(headerBits[1:], ":")
			}
		} else {
			if line == "" {
				continue
			}

			cEntry.Target = line
			out.Entries = append(out.Entries, cEntry)
			cEntry = HLSPlaylistEntry{ Headers: map[string]string{} }
		}
	}

	return out
}