package mining

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/dto"
)

// this whole function (EDIT: is now functions) is yucky to read. I hate it.
func getSignaturesAndDistrubute(signBlock *dto.NodeSignatures) error {
	// just do stuff for running locally for now

	// TBD. Can't get them to talk to each other from the terminal yet.
	localHostPorts := []string{":8080", ":8081", ":8082", ":8083", ":8084", ":8085", ":8086"}
	// activeLocalHostPorts := make([]string, 0)

	// for _, port := range localHostPorts {
	// 	url := fmt.Sprintf("http://127.0.0.1%s/healthcheck", port)
	// 	_, err := http.Get(url)
	// 	if err == nil {
	// 		continue
	// 	}

	// 	activeLocalHostPorts = append(activeLocalHostPorts, port)
	// }

	accumulateSignatures, err := getSignatures(localHostPorts, signBlock)
	if err != nil {
		return err
	}

	// simulate network delay in getting the signatures back from the other nodes
	// time.Sleep(20 * time.Second)

	signBlock.Signatures = append(signBlock.Signatures, accumulateSignatures...)

	err = distribute(localHostPorts, signBlock)
	if err != nil {
		return err
	}

	return nil
}

func getSignatures(localHostPorts []string, signBlock *dto.NodeSignatures) ([]*dto.NodeSignature, error) {
	// get signatures
	accumulateSignatures := make([]*dto.NodeSignature, 0)
	countRejected := 0
	var lastFoundResponseErr error

	for _, port := range localHostPorts {
		signBlockBytes, err := json.Marshal(signBlock)
		if err != nil {
			log.Println("could not marshal request for signing", err)
			countRejected++
			continue
		}

		reqBody := bytes.NewBuffer(signBlockBytes)

		useURL := fmt.Sprintf("http://127.0.0.1%s/block-sign", port)
		log.Println("getting signature from", useURL)
		resp, err := http.DefaultClient.Post(useURL, "application/json", reqBody)
		if err != nil {
			log.Println("not signed by node", err)
			countRejected++
			continue
		}

		respBodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastFoundResponseErr = fmt.Errorf(string(respBodyBytes))
			// log.Println("not signed by node", string(respBodyBytes), "status:", resp.StatusCode, err)
			countRejected++
			continue
		}
		resp.Body.Close()

		newlySignedBlock := &dto.NodeSignatures{}
		err = json.Unmarshal(respBodyBytes, newlySignedBlock)
		if err != nil {
			log.Println("not signed by node", err)
			countRejected++
			continue
		}

		if len(newlySignedBlock.Signatures) < 2 {
			// looks like they didn't actually sign it. think think think.
			log.Println("not actually signed by node only have 1 or fewer signatures")
			countRejected++
			continue
		}

		// the response signature should always be the second signature and ours should always be the first
		accumulateSignatures = append(accumulateSignatures, newlySignedBlock.Signatures[1])
	}

	if countRejected*100/len(localHostPorts) != 100 {
		return nil, lastFoundResponseErr
	}

	return accumulateSignatures, nil
}

func distribute(localHostPorts []string, signBlock *dto.NodeSignatures) error {
	// get % accepted. countReject * 100 / countSent
	countRejected := 0
	var lastFoundResponseErr error
	for _, port := range localHostPorts {
		signBlockBytes, err := json.Marshal(signBlock)
		if err != nil {
			log.Println("not sent to node", port, err.Error())
			continue
		}

		reqBody := bytes.NewBuffer(signBlockBytes)

		useURL := fmt.Sprintf("http://127.0.0.1%s/block", port)
		resp, err := http.DefaultClient.Post(useURL, "application/json", reqBody)
		if err != nil {
			log.Println("not accepted by node", port, err.Error())
			countRejected++
			continue
		}

		if resp.StatusCode != http.StatusOK {
			respBodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				log.Println("not accepted by node", port, err.Error())
				continue
			}
			resp.Body.Close()
			lastFoundResponseErr = fmt.Errorf(string(respBodyBytes))
			// log.Println(string(respBodyBytes))
			continue
		}
		resp.Body.Close()
	}

	// node will only consider its block denied in its own chain if 100% of nodes accept request
	// TODO: when critical mass number of nodes are found, use 70% for acceptance, rejection over 30% for considering block denied.
	// if your malicious goal was to halt the blockchain network, you only need to do proof of work
	// and get accepted onto the chain for 30% of nodes
	// and then you can auto reject to sign or accept blocks. Now none of the nodes can build onto the blockchain.
	// you could decide to only reject when the nodes are not yours, but you still have to get verified by 70% to write a block.
	// so it is easier to attack with a halt than to write bad blocks
	if countRejected*100/len(localHostPorts) != 100 {
		return lastFoundResponseErr
	}

	return nil
}
