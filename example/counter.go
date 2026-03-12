// package main

// import (
// 	"crypto/hmac"
// 	"crypto/sha256"
// 	"encoding/binary"
// 	"errors"
// 	"fmt"
// 	"strings"
// 	"sync/atomic"
// 	"time"
// )

// const (
// 	alphabet      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
// 	base          = int64(62)
// 	codeLen       = 7
// 	maxID         = int64(3_521_614_606_208) // 62^7
// 	secretKey     = "change-this-to-a-long-random-secret-in-production!"
// 	feistelLeft   = int64(1_876_614)
// 	feistelRight  = int64(1_876_614)
// 	feistelRounds = 6
// )

// func hmacRound(val int64, round int) int64 {
// 	key := []byte(secretKey)
// 	buf := make([]byte, 9)
// 	binary.LittleEndian.PutUint64(buf[:8], uint64(val))
// 	buf[8] = byte(round)
// 	mac := hmac.New(sha256.New, key)
// 	mac.Write(buf)
// 	h := mac.Sum(nil)
// 	return int64(binary.LittleEndian.Uint64(h[:8]) & 0x7FFFFFFFFFFFFFFF)
// }

// func feistelEncrypt(id int64) int64 {
// 	L := id / feistelRight
// 	R := id % feistelRight
// 	for i := 0; i < feistelRounds; i++ {
// 		newL := R
// 		f := hmacRound(R, i) % feistelLeft
// 		newR := (L + f) % feistelLeft
// 		L, R = newL, newR
// 	}
// 	return L*feistelRight + R
// }

// func feistelDecrypt(enc int64) int64 {
// 	L := enc / feistelRight
// 	R := enc % feistelRight
// 	for i := feistelRounds - 1; i >= 0; i-- {
// 		newR := L
// 		f := hmacRound(L, i) % feistelLeft
// 		newL := (R - f + feistelLeft) % feistelLeft
// 		L, R = newL, newR
// 	}
// 	return L*feistelRight + R
// }

// func shuffle(id int64) int64 {
// 	x := id
// 	for {
// 		x = feistelEncrypt(x)
// 		if x < maxID {
// 			return x
// 		}
// 	}
// }

// func unshuffle(scrambled int64) int64 {
// 	x := scrambled
// 	for {
// 		x = feistelDecrypt(x)
// 		if x < maxID {
// 			return x
// 		}
// 	}
// }

// func encodeBase62(num int64) string {
// 	if num == 0 {
// 		return string(alphabet[0])
// 	}
// 	result := []byte{}
// 	for num > 0 {
// 		result = append([]byte{alphabet[num%base]}, result...)
// 		num /= base
// 	}
// 	return string(result)
// }

// func decodeBase62(s string) (int64, error) {
// 	var result int64
// 	for _, c := range s {
// 		idx := strings.IndexRune(alphabet, c)
// 		if idx < 0 {
// 			return 0, fmt.Errorf("ky tu khong hop le: %c", c)
// 		}
// 		result = result*base + int64(idx)
// 	}
// 	return result, nil
// }

// func IDToCode(id int64) (string, error) {
// 	if id < 0 || id >= maxID {
// 		return "", fmt.Errorf("id %d out of range", id)
// 	}
// 	scrambled := shuffle(id)
// 	code := encodeBase62(scrambled)
// 	if len(code) < codeLen {
// 		code = strings.Repeat(string(alphabet[0]), codeLen-len(code)) + code
// 	}
// 	return code, nil
// }

// func CodeToID(code string) (int64, error) {
// 	if len(code) != codeLen {
// 		return 0, fmt.Errorf("code phai du %d ky tu", codeLen)
// 	}
// 	num, err := decodeBase62(code)
// 	if err != nil {
// 		return 0, err
// 	}
// 	if num >= maxID {
// 		return 0, errors.New("code khong hop le")
// 	}
// 	return unshuffle(num), nil
// }

// type URLStore struct {
// 	codeToURL map[string]string
// 	urlToCode map[string]string
// 	counter   int64
// }

// func NewURLStore() *URLStore {
// 	return &URLStore{
// 		codeToURL: make(map[string]string),
// 		urlToCode: make(map[string]string),
// 	}
// }

// func (s *URLStore) Shorten(longURL string) (string, error) {
// 	if code, ok := s.urlToCode[longURL]; ok {
// 		return code, nil
// 	}
// 	id := atomic.AddInt64(&s.counter, 1) - 1
// 	if id >= maxID {
// 		return "", errors.New("da dat gioi han URL")
// 	}
// 	code, err := IDToCode(id)
// 	if err != nil {
// 		return "", err
// 	}
// 	s.codeToURL[code] = longURL
// 	s.urlToCode[longURL] = code
// 	return code, nil
// }

// func (s *URLStore) Resolve(code string) (string, error) {
// 	url, ok := s.codeToURL[code]
// 	if !ok {
// 		return "", errors.New("khong tim thay URL")
// 	}
// 	return url, nil
// }

// func truncate(s string, n int) string {
// 	if len(s) <= n {
// 		return s
// 	}
// 	return s[:n-3] + "..."
// }

// func main() {
// 	store := NewURLStore()

// 	urls := []string{
// 		"https://example.com/very/long/url/that/needs/shortening",
// 		"https://github.com/golang/go/blob/master/README.md",
// 		"https://docs.anthropic.com/claude/reference/getting-started",
// 		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
// 		"https://news.ycombinator.com/item?id=123456789",
// 	}

// 	fmt.Println("╔══════════════════════════════════════════════════════════╗")
// 	fmt.Println("║              URL Shortener — Feistel FPE                 ║")
// 	fmt.Println("║  Capacity: 3,521,614,606,208 URLs (62^7)                 ║")
// 	fmt.Println("╚══════════════════════════════════════════════════════════╝")

// 	fmt.Println("\n📎 Rut gon URLs:")
// 	type entry struct{ code, long string }
// 	var entries []entry

// 	for _, url := range urls {
// 		code, err := store.Shorten(url)
// 		if err != nil {
// 			fmt.Printf("  ERROR: %v\n", err)
// 			continue
// 		}
// 		entries = append(entries, entry{code, url})
// 		fmt.Printf("  %s\n  -> https://sho.rt/%s\n\n", truncate(url, 55), code)
// 	}

// 	fmt.Println("✅ Kiem tra resolve:")
// 	allOK := true
// 	for _, e := range entries {
// 		resolved, err := store.Resolve(e.code)
// 		if err != nil || resolved != e.long {
// 			fmt.Printf("  FAIL: %s\n", e.code)
// 			allOK = false
// 		} else {
// 			fmt.Printf("  OK  sho.rt/%s -> %s\n", e.code, truncate(resolved, 45))
// 		}
// 	}
// 	if allOK {
// 		fmt.Println("\n  Tat ca URL resolve dung!")
// 	}

// 	fmt.Println("\n🔀 Sequential IDs -> Non-sequential codes:")
// 	fmt.Println("  ID  | Short Code")
// 	fmt.Println("  ----+-----------")
// 	for i := int64(0); i < 15; i++ {
// 		code, _ := IDToCode(i)
// 		fmt.Printf("  %3d | %s\n", i, code)
// 	}

// 	fmt.Println("\n🔄 Kiem tra bijection (1000 IDs):")
// 	bijectionOK := true
// 	for i := int64(0); i < 1000; i++ {
// 		code, _ := IDToCode(i)
// 		decoded, _ := CodeToID(code)
// 		if decoded != i {
// 			fmt.Printf("  FAIL: ID %d -> %s -> %d\n", i, code, decoded)
// 			bijectionOK = false
// 			break
// 		}
// 	}
// 	if bijectionOK {
// 		fmt.Println("  OK: 1,000 ID encode/decode chinh xac — khong collision!")
// 	}

// 	fmt.Println("\n⚡ Performance:")
// 	n := 10_000
// 	start := time.Now()
// 	for i := int64(0); i < int64(n); i++ {
// 		IDToCode(i)
// 	}
// 	elapsed := time.Since(start)
// 	fmt.Printf("  %d encodes trong %v (%.1f us/op)\n", n, elapsed, float64(elapsed.Microseconds())/float64(n))

//		fmt.Println("\n📐 Production DB Schema:")
//		fmt.Println(`  CREATE TABLE urls (
//	    id         BIGSERIAL PRIMARY KEY,
//	    code       CHAR(7)  NOT NULL UNIQUE,
//	    long_url   TEXT     NOT NULL,
//	    created_at TIMESTAMPTZ DEFAULT NOW(),
//	    hit_count  BIGINT DEFAULT 0
//	  );`)
//	}
package example

func test() {}
