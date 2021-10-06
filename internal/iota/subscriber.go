package iota

/*
#cgo CFLAGS: -I./include -DIOTA_STREAMS_CHANNELS_CLIENT
#cgo LDFLAGS: -L./include -liota_streams_c
#include <channels.h>
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	logInterface "github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"github.com/project-alvarium/stream-subscriber/internal/interfaces"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
	"unsafe"
)

/* The Subscriber has been implemented inline here because I'm not thinking favorably about putting stream subscription
   into the SDK. The SDK interface should be kept simple, and its responsibility limited to Annotations.

	However it's conceivable we might have a module -- provider-streams -- that would contain this inline IOTA integration
    as well as other streaming platforms like MQTT, Kafka, etc.

	I used the Iota Publisher inside the SDK and also the RustAuthorConsole as examples informing this work.
 */

// For randomized seed generation
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const payloadLength = 1024

type iotaSubscriber struct {
	cfg        config.IotaStreamConfig
	logger     logInterface.Logger
	keyload    *C.message_links_t // The Keyload indicates a key needed by the publisher to send messages to the stream
	subscriber *C.subscriber_t    // The publisher is actually subscribed to the stream
	seed       string
	key        string
}

func NewIotaSubscriber(cfg config.IotaStreamConfig, logger logInterface.Logger, key string) interfaces.StreamSubscriber {
	bytes := make([]byte, 64)
	rand.Seed(time.Now().UnixNano())
	for i := range bytes {
		bytes[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	seed := string(bytes)
	logger.Write(logging.DebugLevel, fmt.Sprintf("generated streams seed %s", seed))
	return &iotaSubscriber{
		cfg:    cfg,
		logger: logger,
		seed:   seed,
		key:    key,
	}
}

func (s *iotaSubscriber) Connect() error {
	// Generate Transport client
	transport := C.transport_client_new_from_url(C.CString(s.cfg.TangleNode.Uri()))
	s.logger.Write(logging.DebugLevel, fmt.Sprintf("transport established %s", s.cfg.TangleNode.Uri()))

	// Generate Subscriber instance
	cErr := C.sub_new(&s.subscriber, C.CString(s.seed), C.CString(s.cfg.Encoding), payloadLength, transport)
	s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
	s.logger.Write(logging.DebugLevel, fmt.Sprintf("subscriber established seed=%s", s.seed))

	// Process announcement message
	rawId, err := s.getAnnouncementId(s.cfg.Provider.Uri())
	s.logger.Write(logging.DebugLevel, fmt.Sprintf("Got announcement"))
	if err != nil {
		return err
	}

	var pskid *C.psk_id_t
	// Store psk
	cErr = C.sub_store_psk(&pskid, s.subscriber, C.CString(s.key))
	s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
	if cErr == C.ERR_OK {
		address := C.address_from_string(C.CString(rawId))
		cErr = C.sub_receive_announce(s.subscriber, address)
		s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
		if cErr == C.ERR_OK {
			// Fetch sub link and pk for subscription
			var subLink *C.address_t
			var subPk *C.public_key_t

			cErr = C.sub_send_subscribe(&subLink, s.subscriber, address)
			s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
			if cErr == C.ERR_OK {
				cErr = C.sub_get_public_key(&subPk, s.subscriber)
				s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
				if cErr == C.ERR_OK {
					subIdStr := C.get_address_id_str(subLink)
					subPkStr := C.public_key_to_string(subPk)

					s.logger.Write(logging.DebugLevel, fmt.Sprintf("send subscription request %s", C.GoString(subIdStr)))
					r := subscriptionRequest{
						MsgId: C.GoString(subIdStr),
						Pk:    C.GoString(subPkStr),
					}
					body, _ := json.Marshal(&r)
					sendSubscriptionIdToAuthor(s.cfg.Provider.Uri(), body)
					s.logger.Write(logging.DebugLevel, "subscription request sent")

					C.drop_str(subIdStr)
					C.drop_str(subPkStr)
					return nil
				}
			}
		}
	}
	return errors.New("failed to connect publisher")
}

func (s *iotaSubscriber) Read() error {
	var messages *C.unwrapped_messages_t
	cErr := C.sub_sync_state(&messages, s.subscriber)
	//defer C.drop_unwrapped_messages(messages)

	if cErr == C.ERR_OK {
		count := int(C.get_payloads_count(messages))
		idx := 0
		for idx < count {
			msg := C.get_indexed_payload(messages, C.size_t(idx))
			out := C.GoBytes(unsafe.Pointer(msg.masked_payload.ptr), C.int(msg.masked_payload.size))
			fmt.Println(msg.masked_payload.ptr, msg.masked_payload.size)
			// In this example the message is simply written to a string. However if you want to unmarshal the bytes
			// into a type, you will have to pass the []byte out via a channel rather than calling json.Unmarshal() here.
			// Doing the latter will result in an error related to an invalid pointer.
			fmt.Println(fmt.Sprintf("Message -- len:%v txt:%s", len(out), string(out)))
			C.drop_payloads(msg)
			idx++
		}
	} else {
		return errors.New(get_error(cErr))
	}
	return nil
}


func (s *iotaSubscriber) Close() error {
	C.sub_drop(s.subscriber)
	return nil
}

func (s *iotaSubscriber) getAnnouncementId(url string) (string, error) {
	type announcementResponse struct {
		AnnouncementId string `json:"announcement_id"`
	}

	s.logger.Write(logging.DebugLevel, fmt.Sprintf("GET %s/get_announcement_id", url))
	resp, err := http.Get(url + "/get_announcement_id")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	s.logger.Write(logging.DebugLevel, fmt.Sprintf("announcement response - %s", string(bodyBytes)))
	var annResp announcementResponse
	if err := json.Unmarshal(bodyBytes, &annResp); err != nil {
		return "", err
	}
	return annResp.AnnouncementId, nil
}

func sendSubscriptionIdToAuthor(url string, body []byte) error {
	client := http.Client{}
	data := bytes.NewReader(body)
	req, err := http.NewRequest("POST", url+"/subscribe", data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type subscriptionRequest struct {
	MsgId string `json:"msgid"`
	Pk    string `json:"pk"`
}

func get_error(err C.err_t) string {
	var e = "Unknown Error"
	switch err {
	case C.ERR_OK:
		e = "Operation completed successfully"
	case C.ERR_OPERATION_FAILED:
		e = "Streams operation failed to complete successfully"
	case C.ERR_NULL_ARGUMENT:
		e = "The function was passed a null argument"
	case C.ERR_BAD_ARGUMENT:
		e = "The function was passed a bad argument"
	}
	return e
}
