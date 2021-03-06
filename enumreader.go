// Copyright 2011 The GoGL Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.mkd file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// TODO: better regexps
var (
	enumCommentLineRE = regexp.MustCompile("^#.*")
	enumCategoryRE    = regexp.MustCompile("^([_0-9A-Za-z]+)[ \\t]+enum:")
	enumRE            = regexp.MustCompile("^([_0-9A-Za-z]+)[ \\t]*=[ \\t]*([\\-_0-9A-Za-z]+)")
	enumPassthruRE    = regexp.MustCompile("^passthru:.*")
	enumUseRE         = regexp.MustCompile("^use[ \\t]+([_0-9A-Za-z]+)[ \\t]+([_0-9A-Za-z]+)")
)

func ReadEnumsFromFile(name string) (EnumCategories, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ReadEnums(file)
}

func ReadEnums(r io.Reader) (EnumCategories, error) {
	categories := make(EnumCategories)
	deferredUseEnums := make(map[string]map[string]string)
	currentCategory := ""
	br := bufio.NewReader(r)

	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		line = strings.Trim(line, "\t\n\r ")

		if len(line) == 0 || enumCommentLineRE.MatchString(line) || enumPassthruRE.MatchString(line) {
			//fmt.Printf("Empty or comment line %s\n", line)
			continue
		}

		if category := enumCategoryRE.FindStringSubmatch(line); category != nil {
			//fmt.Printf("%v\n", category[1])
			currentCategory = category[1]
			categories[currentCategory] = make(Enums)
		} else if enum := enumRE.FindStringSubmatch(line); enum != nil {
			//fmt.Printf("%v %v\n", enum[1], enum[2])
			if strings.HasPrefix(enum[2], "GL_") {
				//fmt.Printf("Lookup %s in %s\n", enum[2][3:], enum[1])
				ok, val := categories.LookUpDefinition(enum[2][3:])
				if ok {
					categories[currentCategory][enum[1]] = val
				} else {
					fmt.Fprintf(os.Stderr, "ERROR: Unable to find %s.\n", enum[2][3:])
				}
			} else if strings.HasPrefix(enum[2], "GLX_") {
				//fmt.Printf("Lookup %s in %s\n", enum[2][3:], enum[1])
				ok, val := categories.LookUpDefinition(enum[2][4:])
				if ok {
					categories[currentCategory][enum[1]] = val
				} else {
					fmt.Fprintf(os.Stderr, "ERROR: Unable to find %s.\n", enum[2][4:])
				}
			} else if strings.HasSuffix(enum[2], "u") {
				categories[currentCategory][enum[1]] = enum[2][:len(enum[2])-1]
			} else if strings.HasSuffix(enum[2], "ull") {
				categories[currentCategory][enum[1]] = enum[2][:len(enum[2])-3]
			} else {
				categories[currentCategory][enum[1]] = enum[2]
			}
		} else if use := enumUseRE.FindStringSubmatch(line); use != nil {
			//fmt.Printf("%v %v\n", use[1], use[2])
			if deferredUseEnums[currentCategory] == nil {
				deferredUseEnums[currentCategory] = make(map[string]string)
			}
			deferredUseEnums[currentCategory][use[2]] = use[1]
		} else {
			fmt.Fprintf(os.Stderr, "WARNING: Unable to parse line: '%s' (Ignoring)\n", line)
		}
	}

	for category, enums := range deferredUseEnums {
		for name, referencedCategory := range enums {
			if dereference, ok := categories[referencedCategory][name]; ok {
				categories[category][name] = dereference
			} else if dereference, ok := categories[referencedCategory+"_DEPRECATED"][name]; ok {
				categories[category][name] = dereference
			} else {
				fmt.Fprintf(os.Stderr, "WARNING: Failed to dereference %v: \"use %v %v\"\n", category, referencedCategory, name)
			}
		}
	}

	return categories, nil
}
