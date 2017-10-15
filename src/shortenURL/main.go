package main

import (
	"fmt"
	"github.com/speps/go-hashids"
	"github.com/gorilla/mux"
	"github.com/go-redis/redis"
	"log"
	"net/http"
	"shortenURL/hashid"
	"time"
)

// PostURL converts the long URL to a short URL
func PostURL(w http.ResponseWriter, r *http.Request) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	urlCounter, err := client.Incr("id").Result()
	err = client.SetNX(string(urlCounter), r.URL.Path[1:], 0).Err()
	if err != nil {
		fmt.Printf("Failed to store URL")
		//Instead of panic we should wait for Redis server to come back up
		panic(err)
	}
	shortURL := hashid.EncodeID(int(urlCounter))

	//Initialize hit counts. The last argument sets the expiration time
	client.SetNX(fmt.Sprintf("%s.d", shortURL),0, time.Hour*24)
	client.SetNX(fmt.Sprintf("%s.w", shortURL), 0, time.Hour*24*7)
	client.SetNX(fmt.Sprintf("%s.a", shortURL), 0, 0)

	fmt.Fprintf(w, "Short URL = http://localhost:8000/%s\n", shortURL)
	fmt.Fprintf(w, "Hit Count Stats = http://localhost:8000/%s/count\n", shortURL)
}

// GetCount retrieves hitcounts for an URL in the last 24 hrs, 1 week and all time
func GetCount(w http.ResponseWriter, r *http.Request){
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	urlPath := r.URL.Path[1:7]
	dayCount,err := client.Get(fmt.Sprintf("%s.d", urlPath)).Result()
	if err != nil {
		//The key may have been expired
		dayCount = "0"
	}
	weekCount,err := client.Get(fmt.Sprintf("%s.w", urlPath)).Result()
	if err != nil {
		//The key may have been expired
		weekCount = "0"
	}
	allCount,err := client.Get(fmt.Sprintf("%s.a", urlPath)).Result()
	if err != nil {
		//Ideally this should never happen
		panic(err)
	}
	fmt.Fprintf(w, "Daily hitcount = %s\nWeekly hitcount = %s\nAll time hitcount = %s\n",
		dayCount,weekCount, allCount)
}

// VisitURL redirects to a long URL from a short URL
func VisitURL(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path[1:]
	id := uint64(hashid.DecodeID(urlPath))
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	longURL, err := client.Get(string(id)).Result()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, fmt.Sprintf("http://%s",longURL), http.StatusFound)

	// Increment hit counts
	client.Incr(fmt.Sprintf("%s.d",urlPath))
	client.Incr(fmt.Sprintf("%s.w", urlPath))
	client.Incr(fmt.Sprintf("%s.a", urlPath))
}

// ExpireWatcher goroutine subscribes for expired events and reinitializes the counter
func ExpireWatcher() {
	client := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"shard1": ":6379",
		},
	})
	pubsub := client.PSubscribe("__keyevent@0__:expired")
	defer pubsub.Close()
	for {
		msg,err := pubsub.ReceiveMessage()
		if err != nil {
			fmt.Println("Error receiving subscribed event")
			return
		}
		key := msg.Payload
		stat := key[len(key)-2:]
		switch stat {
			case ".d":
				client.SetNX(key,0, time.Hour*24)
			case ".w":
				client.SetNX(key, 0, time.Hour*24*7)
			default:
				fmt.Println("Invalid event")

		}
	}
}

// main function
func main() {
	router := mux.NewRouter()
	hashid.Hashes = hashids.NewData()
	hashid.Hashes.Salt = "Cloudflare"
	hashid.Hashes.MinLength = 6
	router.HandleFunc("/{url}", PostURL).Methods("POST")
	router.HandleFunc("/{url}/count", GetCount).Methods("GET")
	router.HandleFunc("/{short_url}", VisitURL).Methods("GET")
	//Initialize a goroutine to watch for expired events
	go ExpireWatcher()
	log.Fatal(http.ListenAndServe(":8000", router))
}

