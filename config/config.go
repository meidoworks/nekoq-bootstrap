package config

import (
	"io"
	"bufio"
	"strings"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
)

type ConfigFileData map[string]map[string]string

const (
	_SECTION_START    = 1
	_SECTION_CONTENTS = 2
)

func Load(r io.Reader) (config ConfigFileData, err error) {
	reader := bufio.NewReader(r)

	config = make(ConfigFileData)

	state := _SECTION_START
	var sectionMap map[string]string
	var allowDupSection = false
	var line string
	for {
		line, err = reader.ReadString('\n')

		line = strings.TrimSpace(line)
		if len(line) > 0 && !isComment(line) {
			fmt.Println(line)
			switch state {
			case _SECTION_START:
				if !isSectionStart(line) {
					return config, errors.New("not section start. line=" + line)
				}
				sectionMap, err = getSection(config, line, allowDupSection)
				if err != nil {
					return
				}
				state = _SECTION_CONTENTS
			case _SECTION_CONTENTS:
				if isSectionStart(line) {
					if !isSectionStart(line) {
						return config, errors.New("not section start. line=" + line)
					}
					sectionMap, err = getSection(config, line, allowDupSection)
					if err != nil {
						return
					}
					state = _SECTION_CONTENTS
				} else {
					err = processKeyValue(sectionMap, line)
					if err != nil {
						return
					}
					state = _SECTION_CONTENTS
				}
			default:
				return config, errors.New("unknown state=" + strconv.Itoa(state))
			}
		}

		if err == nil {
			continue
		} else if err == io.EOF {
			err = nil
			return
		} else {
			return
		}
	}
}

func processKeyValue(sectionMap map[string]string, line string) error {
	idx := strings.Index(line, "=")
	if idx == -1 || idx == 0 {
		return errors.New("key value format illegal. line=" + line)
	}
	key := line[0:idx]
	// only key to trim
	key = strings.TrimSpace(key)
	var value string
	if idx == len(line)-1 {
		value = ""
	} else {
		value = line[idx+1:]
	}
	sectionMap[key] = value
	return nil
}

func getSection(data ConfigFileData, s string, allowDup bool) (map[string]string, error) {
	key := s[1 : len(s)-1]
	// only key to trim
	key = strings.TrimSpace(key)
	sectionMap, ok := data[key]
	if ok && !allowDup {
		return sectionMap, errors.New("duplicated section=" + key)
	}
	result := make(map[string]string)
	data[key] = result
	return result, nil
}

func isSectionStart(s string) bool {
	return s[0] == '[' && s[len(s)-1] == ']'
}

func isComment(s string) bool {
	return s[0] == '#'
}
