package transfer

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
	clientRpc "github.com/openweb3/go-rpc-provider"
	"github.com/sirupsen/logrus"
)

// ======= Record process state =======

type ProcessState struct {
	Space             types.SpaceType
	Sender            *cfxaddress.Address
	ReceiverListHash  string
	TokenSymbol       string
	TokenAddress      *cfxaddress.Address
	SendingStartIdx   int
	SendingBatchElems []clientRpc.BatchElem
}

func clearCacheFile() {
	os.Remove(resultPath)
}

func loadProcessState() ProcessState {
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		return ProcessState{}
	}

	content, e := ioutil.ReadFile(resultPath)
	util.OsExitIfErr(e, "Read result content error")

	if len(content) == 0 {
		return ProcessState{}
	}

	var ps ProcessState
	e = json.Unmarshal(content, &ps)
	util.OsExitIfErr(e, "Failed to unmarshal process state")

	return ps
}

var m sync.Mutex

func (s *ProcessState) UnmarshalJSON(data []byte) error {
	type tmpType struct {
		Sender            *cfxaddress.Address
		ReceiverListHash  string
		TokenSymbol       string
		TokenAddress      *cfxaddress.Address
		SendingStartIdx   int
		SendingBatchElems []struct {
			Method string
			Args   []interface{}
			Result *string
			Error  *utils.RpcError
		}
	}

	t := tmpType{}
	if e := json.Unmarshal(data, &t); e != nil {
		return e
	}

	s.Sender = t.Sender
	s.ReceiverListHash = t.ReceiverListHash
	s.SendingStartIdx = t.SendingStartIdx
	s.TokenSymbol = t.TokenSymbol
	s.TokenAddress = t.TokenAddress
	s.SendingBatchElems = make([]clientRpc.BatchElem, len(t.SendingBatchElems))
	for i, v := range t.SendingBatchElems {
		s.SendingBatchElems[i].Method = v.Method
		s.SendingBatchElems[i].Args = v.Args
		s.SendingBatchElems[i].Result = v.Result
		if v.Error != nil {
			s.SendingBatchElems[i].Error = v.Error
		}
	}
	return nil
}

func (s *ProcessState) save() {
	m.Lock()
	defer m.Unlock()

	f, e := os.OpenFile(resultPath, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0777)
	util.OsExitIfErr(e, "Failed to create file")
	defer f.Close()

	j, e := json.MarshalIndent(s, "", "  ")
	util.OsExitIfErr(e, "Failed to marshal")

	_, e = f.Write(j)
	util.OsExitIfErr(e, "Failed to save state")
}

func (s *ProcessState) refreshSpaceAndSave(space types.SpaceType) {
	if s.Space != space {
		s.Space = space
		s.clearSendingsAndSave()
	}
}

func (s *ProcessState) refreshSenderAndSave(from *cfxaddress.Address) {
	if s.Sender == nil {
		s.Sender = from
		s.save()
		return
	}

	if from.String() != s.Sender.String() {
		s.clearSendingsAndSave()
		s.Sender = from
		s.save()
	}
}

func (s *ProcessState) refreshReceiversAndSave(receverList []Receiver) {
	// fmt.Printf("refreshByReceivers, ProcessState,%+v\n", s)
	r, e := json.Marshal(receverList)
	util.OsExitIfErr(e, "Failed to marshal receiver list")
	newReceiverHash := fmt.Sprintf("%x", md5.Sum(r))
	if s.ReceiverListHash != newReceiverHash {
		logrus.Debugf("%v != %v\n", s.ReceiverListHash, newReceiverHash)
		*s = ProcessState{}
	}
	s.ReceiverListHash = newReceiverHash
	s.save()
}

func (s *ProcessState) setSelectTokenAndSave(tokenSymbol string, tokenAddress *cfxaddress.Address) {
	if s.TokenSymbol != tokenSymbol || s.TokenAddress != tokenAddress {
		logrus.Debugf("refresh select token,%v,%v\n", s.TokenSymbol, s.TokenAddress)
		s.TokenSymbol = tokenSymbol
		s.TokenAddress = tokenAddress
		// reset last sending info
		s.SendingStartIdx = 0
		s.SendingBatchElems = nil
		s.save()
	}
}

func (s *ProcessState) setSendingsAndSave(sendingStartIdx int, rpcBatchElems []clientRpc.BatchElem) {
	logrus.Debugf("saveSendings, sendingStartIdx %+v\n", sendingStartIdx)
	s.SendingBatchElems = rpcBatchElems
	s.SendingStartIdx = sendingStartIdx
	s.save()
}

func (s *ProcessState) clearSendingsAndSave() {
	s.setSendingsAndSave(0, nil)
}
