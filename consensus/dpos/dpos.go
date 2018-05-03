package dpos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	. "github.com/iost-official/prototype/consensus/common"
	. "github.com/iost-official/prototype/p2p"

	"github.com/iost-official/prototype/common"
	"github.com/iost-official/prototype/core"
)

type DPoS struct {
	core.Member
	BlockCache
	Router
	GlobalStaticProperty
	GlobalDynamicProperty

	//测试用，保存投票状态，以及投票消息内容的缓存
	votedStats map[string][]string
	infoCache  [][]byte

	ExitSignal chan bool
	chTx       chan core.Request
	chBlock    chan core.Request
}

func NewDPoS(mb core.Member, bc core.BlockChain /*, network core.Network*/) (*DPoS, error) {
	// Member初始化
	p := DPoS{}
	p.Member = mb
	p.BlockCache = NewBlockCache(bc, 6)

	var err error
	p.Router, err = RouterFactory("base")
	if err != nil {
		return nil, err
	}

	//	Tx chan init
	p.chTx, err = p.Router.FilteredChan(Filter{
		WhiteList:  []core.Member{},
		BlackList:  []core.Member{},
		RejectType: []ReqType{},
		AcceptType: []ReqType{
			ReqPublishTx,
			ReqTypeVoteTest, // Only for test
		}})
	if err != nil {
		return nil, err
	}

	//	Block chan init
	p.chBlock, err = p.Router.FilteredChan(Filter{
		WhiteList:  []core.Member{},
		BlackList:  []core.Member{},
		RejectType: []ReqType{},
		AcceptType: []ReqType{ReqNewBlock}})
	if err != nil {
		return nil, err
	}

	p.initGlobalProperty(p.Member, []string{"id0", "id1", "id2", "id3"})
	return &p, nil
}

func (p *DPoS) initGlobalProperty(mb core.Member, witnessList []string) {
	p.GlobalStaticProperty = NewGlobalStaticProperty(mb, witnessList)
	p.GlobalDynamicProperty = NewGlobalDynamicProperty()
}

func (p *DPoS) Run() {
	//go p.blockLoop()
	//go p.scheduleLoop()
	p.genBlock(p.Member, core.Block{})
}

func (p *DPoS) Stop() {
	close(p.chTx)
	close(p.chBlock)
	p.ExitSignal <- true
}

func (p *DPoS) Genesis(initTime Timestamp, hash []byte) error {
	return nil
}

func (p *DPoS) txListenLoop() {
	for {
		req, ok := <-p.chTx
		if !ok {
			return
		}
		if req.ReqType == ReqTypeVoteTest {
			p.AddWitnessMsg(req)
			continue
		}
		var tx core.Tx
		tx.Decode(req.Body)
		p.Router.Send(req)
		if VerifyTxSig(tx) {
			// Add to tx pool or recorder
		}
	}
}

func (p *DPoS) blockLoop() {
	//收到新块，验证新块，如果验证成功，更新DPoS全局动态属性类并将其加入block cache，再广播
	verifier := func(blk *core.Block, chain core.BlockChain) bool {
		// verify block head

		if !VerifyBlockHead(blk, chain.Top()) {
			return false
		}

		// verify block witness
		if WitnessOfTime(&p.GlobalStaticProperty, &p.GlobalDynamicProperty, Timestamp{blk.Head.Time}) != blk.Head.Witness {
			return false
		}

		headInfo := generateHeadInfo(blk.Head)
		var signature common.Signature
		signature.Decode(blk.Head.Signature)
		// verify block witness signature
		if !common.VerifySignature(headInfo, signature) {
			return false
		}
		/*
		if !VerifyBlockContent(blk, chain) {
			return false
		}
		*/
		return true
	}

	for {
		req, ok := <-p.chBlock
		if !ok {
			return
		}
		var blk core.Block
		blk.Decode(req.Body)
		err := p.BlockCache.Add(&blk, verifier)
		if err == nil {
			p.GlobalDynamicProperty.Update(&blk.Head)
		}
		ts := Timestamp{blk.Head.Time}
		if ts.After(p.GlobalDynamicProperty.NextMaintenanceTime) {
			p.PerformMaintenance()
		}
	}
}

func (p *DPoS) scheduleLoop() {
	//通过时间判定是否是本节点的slot，如果是，调用产生块的函数，如果不是，设定一定长的timer睡眠一段时间

	for {
		currentTimestamp := GetCurrentTimestamp()
		wid := WitnessOfTime(&p.GlobalStaticProperty, &p.GlobalDynamicProperty, currentTimestamp)
		if wid == p.Member.ID {
			bc := p.BlockCache.LongestChain()
			blk := p.genBlock(p.Member, *bc.Top())
			p.Router.Send(core.Request{Body: blk.Encode()}) //??
		}
		nextSchedule := TimeUntilNextSchedule(&p.GlobalStaticProperty, &p.GlobalDynamicProperty, time.Now().Unix())
		time.Sleep(time.Duration(nextSchedule))
	}
}

func (p *DPoS) genBlock(mb core.Member, lastBlk core.Block) *core.Block {
	/*
		if lastBlk == nil {
			blk := core.Block{Version: 0, Content: make([]byte, 0), Head: core.BlockHead{
				Version:    0,
				ParentHash: lastBlk.Head.BlockHash,
				TreeHash:   make([]byte, 0),
				BlockHash:  make([]byte, 0),
				Info:       make([]byte, 0),
				Number:     0,
				Witness:    mb.ID, // ?
				Time:       GetCurrentTimestamp(),
			}}
			headinfo := generateHeadInfo(blk.Head)
			sig, _ := common.Sign(common.Secp256k1, headinfo, mb.Seckey)
			blk.Head.Signature = sig.Encode()
			return &blk
		}
	*/
	blk := core.Block{Version: 0, Content: make([]byte, 0), Head: core.BlockHead{
		Version:    0,
		ParentHash: lastBlk.Head.BlockHash,
		TreeHash:   make([]byte, 0),
		BlockHash:  make([]byte, 0),
		Info:       encodeDPoSInfo(p.infoCache),
		Number:     lastBlk.Head.Number + 1,
		Witness:    mb.ID,
		Time:       GetCurrentTimestamp().Slot,
	}}
	p.infoCache = [][]byte{}
	headInfo := generateHeadInfo(blk.Head)
	fmt.Println(mb.Seckey)
	sig, _ := common.Sign(common.Secp256k1, headInfo, mb.Seckey)
	blk.Head.Signature = sig.Encode()
	return &blk
}

func generateHeadInfo(head core.BlockHead) []byte {
	var info, numberInfo, versionInfo []byte
	info = make([]byte, 8)
	versionInfo = make([]byte, 4)
	numberInfo = make([]byte, 4)
	binary.BigEndian.PutUint64(info, uint64(head.Time))
	binary.BigEndian.PutUint32(versionInfo, uint32(head.Version))
	binary.BigEndian.PutUint32(numberInfo, uint32(head.Number))
	info = append(info, versionInfo...)
	info = append(info, numberInfo...)
	info = append(info, head.ParentHash...)
	info = append(info, head.TreeHash...)
	info = append(info, head.Info...)
	return info
}

// 测试函数，用来将info和vote消息进行转换，在块被确认时被调用
// TODO:找到适当的时机调用
func decodeDPoSInfo(info []byte) [][]byte {
	votes := bytes.Split(info, []byte("/"))
	return votes
}

// 测试函数，用来将info和vote消息进行转换，在生成块的时候调用
func encodeDPoSInfo(votes [][]byte) []byte {
	var info []byte
	for _, req := range votes {
		info = append(info, req...)
		info = append(info, byte('/'))
	}
	return info
}