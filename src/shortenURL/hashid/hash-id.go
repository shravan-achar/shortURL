package hashid

import (
	"fmt"
	"github.com/speps/go-hashids"
	"strings"
	"strconv"
)

var Hashes *hashids.HashIDData
const chars = "ghijklmnopqrstuvwyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const hexs = "0123456789abcdef"

//EncodeID encodes integer to string using scrambled alphabet
func EncodeID(id int) string {
	h, _ := hashids.NewWithData(Hashes)
	e, _ := h.Encode([]int{id})
	return e
}

//DecodeID decodes string to int using scrambled alphabet
func DecodeID(url string) int {
	h, _ := hashids.NewWithData(Hashes)
	d, _ := h.DecodeWithError(url)
	return d[0]
}

//EncodeID2 is an alternate way to encode integer to a string
func EncodeID2(id int) string {
	var shortURL string
	if id & 1 == 0 {
		//Do Base 46 conversion with letters "[g-z], [A-Z]"
		sURL := make([]byte, 6)
		for id > 0 {
			sURL = append(sURL, chars[id%46])
			id /= 46
		}
		shortURL = string(sURL[:len(sURL)])
	} else {
		//Do Base 16 (Hex) with letters "[0-9], [a-f]"
		shortURL = fmt.Sprintf("%x", id)
	}
	return shortURL
}

//DecodeID2 is an alternate way to decode a string to integer
func DecodeID2(url string) int64 {
	var id int64
	if strings.ContainsAny(url, chars) {
		//Convert it back to base 46 integer
		for i := 0; i < len(url); i++ {
			if url[i] > 'g' && url[i] < 'z' {
				id = id*46 + int64(url[i]) - 'g'
			} else if url[i] > 'A' && url[i] < 'Z' {
				id = id*46 + int64(url[i]) - 'A' + 26
			}
		}
	} else if strings.ContainsAny(url, hexs) {
		//Convert it to decimal
		id,_ = strconv.ParseInt(url, 16, 64)
	}
	return id
}

