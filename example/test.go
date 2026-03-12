// package

// import (
// 	"context"
// 	"crypto/hmac"
// 	"crypto/sha256"
// 	"encoding/binary"
// 	"fmt"
// 	"shortlink/internal/cache"
// 	"strconv"
// )

// const (
// 	_alphabet  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
// 	_base      = int64(62)
// 	_codeLen   = 7
// 	_secretKey = "your-secret-key"
// )

// func shuffle(id int64) int64 {
// 	// Feistel 4 rounds trong không gian 43 bit (>= 62^7 = 42.7 bit)
// 	const half = int64(1 << 21) // 2^21 = 2,097,152
// 	L, R := (id>>21)&0x3FFFFF, id&0x1FFFFF
// 	for i := 0; i < 4; i++ {
// 		buf := make([]byte, 9)
// 		binary.LittleEndian.PutUint64(buf, uint64(R))
// 		buf[8] = byte(i)
// 		mac := hmac.New(sha256.New, []byte(_secretKey))
// 		mac.Write(buf)
// 		f := int64(binary.LittleEndian.Uint64(mac.Sum(nil)[:8])) & 0x7FFFFFFFFFFFFFFF % half
// 		L, R = R, (L+f)%half
// 	}
// 	return (L<<21 | R) % 3_521_614_606_208 // mod 62^7
// }

// func toBase62(num int64) string {
// 	code := make([]byte, 7)
// 	for i := 6; i >= 0; i-- {
// 		code[i] = _alphabet[num%62]
// 		num /= 62
// 	}
// 	return string(code)
// }

// func GenerateShortCode(id int64) string {
// 	return toBase62(shuffle(id))
// }

// func main() {
// 	// snowflake := services.NewSnowflake(1)

// 	redisClient, err := cache.NewRedisClient()
// 	if err != nil {
// 		fmt.Println("Failed to connect to Redis:", err)
// 		return
// 	}
// 	ctx := context.Background()
// 	redisBloom := cache.NewRedisBloom(redisClient)
// 	redisBloom.Reserve(ctx, "user_bloom_key", 0.01, 1200000)

// 	// 3. Tiến hành Add vào Bloom Filter
// 	total := 1_000_000

// 	for i := 0; i < total; i++ {
// 		id := "user_" + strconv.Itoa(i)

// 		err := redisBloom.Add(ctx, "user_bloom_key", id)
// 		if err != nil {
// 			fmt.Printf("Error adding %s: %v\n", id, err)
// 		}
// 	}

// 	// // 4. Kiểm tra và in kết quả
// 	// fmt.Println("--- Kiểm tra các user tồn tại (Kỳ vọng: true) ---")
// 	// for _, id := range existingUserIDs {
// 	// 	exists, _ := redisBloom.Exists(ctx, "user_bloom_key", id)
// 	// 	fmt.Printf("User %s: %v\n", id, exists)
// 	// }

// 	// fmt.Println("--- Kiểm tra các user không tồn tại (Kỳ vọng: false) ---")
// 	// for _, id := range missingUserIDs {
// 	// 	exists, _ := redisBloom.Exists(ctx, "user_bloom_key", id)
// 	// 	fmt.Printf("User %s: %v\n", id, exists)
// 	// }

//		fmt.Println("Done")
//	}
package example

func main() {}
