# Distribute coin API

When user receives an Verifiable Credential(VC) from IDP, user can submit the VC to the Redbelly Network
to gain write permission to the network. However, when user submit this initial request, user MUST have RBNT
to submit the transaction. Therefore, this service, allows IDP to distribute the initial RBNT to user

This service has `/distribute?address=0x70997970C51812dc3A010C7d01b50e0d17dc79C8` to distribute 1 RBNT to
the given address

The team has developed this solution, when this API is called in lower rate, everything work perfectly
However, when it in high load, example 100 requests per seconds, the application start to fail.

## Test Loader

Initially, the team has the following loader `seq/main.go` with the following code

```go
func main() {
	for i := 0; i < 100; i++ {
		resp, err := http.Get("http://localhost:8080/distribute?address=0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("response %v", resp)
	}
}
```

This work perfectly with no error

However, the read world situation is concurrent request `parallel/main.go`

```go
func main() {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://localhost:8080/distribute?address=0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("response %v", resp)
		}()
	}
	wg.Wait()
}
```

## Your tasks

* Improve the API can handle 100 requests per seconds without fail
* The service MUST be able to deliver the coin

## Assumption and Precondition

* No database, can use internal map or any storage for demo purpose

## How to Test

1. Start a local blockchain using hardhat
2. Start the service
3. Make a lot of requests

Each step run in it own console window

### 1. Start local blockchain

```shell
cd node
npm install
npx hardhat node
```

### 2. Start the service

```shell
go run main.go
```

### 3. Start the loader

```shell
go run parallel/main.go
```