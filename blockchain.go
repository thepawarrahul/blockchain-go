package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Block struct {
	Index        int
	Timestamp    string
	BPM          int
	Hash         string
	PreviousHash string
}

type Message struct {
	BMP int
}

var Blockchain []Block

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PreviousHash

	hashFunction := sha256.New()
	hashFunction.Write([]byte(record))

	hashedRecord := hashFunction.Sum(nil)

	return hex.EncodeToString(hashedRecord)
}

func generateBlock(oldBlock Block, BMP int) (Block, error) {
	var newBlock Block

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = time.Now().String()
	newBlock.BPM = BMP
	newBlock.PreviousHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PreviousHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func runServer() error {
	mux := makeMuxRouter()

	httpAddress := os.Getenv("PORT")
	log.Println("Listining on ", os.Getenv("PORT"))

	server := &http.Server{
		Addr:           ":" + httpAddress,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")

	return muxRouter
}

func handleGetBlockchain(responseWriter http.ResponseWriter, request *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(responseWriter, string(bytes))
}

func handleWriteBlock(responseWriter http.ResponseWriter, request *http.Request) {
	var message Message

	decoder := json.NewDecoder(request.Body)

	if err := decoder.Decode(&message); err != nil {
		reponseWithJSON(responseWriter, request, http.StatusBadRequest, request.Body)

		return
	}
	defer request.Body.Close()

	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], message.BMP)

	if err != nil {
		reponseWithJSON(responseWriter, request, http.StatusInternalServerError, message)
		return
	}

	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockChain := append(Blockchain, newBlock)
		replaceChain(newBlockChain)
		spew.Dump(Blockchain)
	}

	reponseWithJSON(responseWriter, request, http.StatusCreated, newBlock)
}

func reponseWithJSON(responseWriter http.ResponseWriter, request *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")

	if err != nil {
		responseWriter.WriteHeader(http.StatusInternalServerError)
		responseWriter.Write([]byte("HTTP 500: Internal Server error"))

		return
	}

	responseWriter.WriteHeader(code)
	responseWriter.Write(response)
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		genesisBlock := Block{0, time.Now().String(), 0, "", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()

	log.Fatal(runServer())
}
