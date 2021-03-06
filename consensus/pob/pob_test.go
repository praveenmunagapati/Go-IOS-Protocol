package pob

import (
	"fmt"
	"testing"

	. "github.com/bouk/monkey"
	. "github.com/golang/mock/gomock"

	"sync"
	"time"

	"os"

	"github.com/iost-official/Go-IOS-Protocol/account"
	"github.com/iost-official/Go-IOS-Protocol/common"
	"github.com/iost-official/Go-IOS-Protocol/consensus/common"
	"github.com/iost-official/Go-IOS-Protocol/core/block"
	"github.com/iost-official/Go-IOS-Protocol/core/message"
	"github.com/iost-official/Go-IOS-Protocol/core/mocks"
	"github.com/iost-official/Go-IOS-Protocol/core/state"
	"github.com/iost-official/Go-IOS-Protocol/core/tx"
	"github.com/iost-official/Go-IOS-Protocol/core/txpool"
	"github.com/iost-official/Go-IOS-Protocol/log"
	"github.com/iost-official/Go-IOS-Protocol/network"
	"github.com/iost-official/Go-IOS-Protocol/network/mocks"
	"github.com/iost-official/Go-IOS-Protocol/vm"
	"github.com/iost-official/Go-IOS-Protocol/vm/lua"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewPoB(t *testing.T) {
	Convey("Test of NewPoB", t, func() {
		mockCtr := NewController(t)
		mockRouter := protocol_mock.NewMockRouter(mockCtr)
		mockBc := core_mock.NewMockChain(mockCtr)
		mockPool := core_mock.NewMockPool(mockCtr)
		mockPool.EXPECT().Copy().Return(mockPool).AnyTimes()
		mockPool.EXPECT().PutHM(Any(), Any(), Any()).AnyTimes().Return(nil)
		mockPool.EXPECT().Flush().AnyTimes().Return(nil)

		network.Route = mockRouter
		guard := Patch(network.RouterFactory, func(_ string) (network.Router, error) {
			return mockRouter, nil
		})
		defer guard.Unpatch()

		heightChan := make(chan message.Message, 1)
		blkChan := make(chan message.Message, 1)
		mockRouter.EXPECT().FilteredChan(Any()).Return(heightChan, nil)
		mockRouter.EXPECT().FilteredChan(Any()).Return(blkChan, nil)

		txChan := make(chan message.Message, 1)
		mockRouter.EXPECT().FilteredChan(Any()).Return(txChan, nil)

		blockChan := make(chan message.Message, 1)
		mockRouter.EXPECT().FilteredChan(Any()).Return(blockChan, nil)

		mockRouter.EXPECT().FilteredChan(Any()).Return(blockChan, nil).AnyTimes()

		blk := block.Block{Content: []tx.Tx{}, Head: block.BlockHead{
			Version:    0,
			ParentHash: []byte("111"),
			Info:       []byte("test"),
			Number:     int64(1),
			Witness:    "11111",
			Time:       1111,
		}}
		blk.Head.TreeHash = blk.CalculateTreeHash()

		var getNumber uint64
		var pushNumber int64
		mockBc.EXPECT().GetBlockByNumber(Any()).Return(nil).AnyTimes()
		mockBc.EXPECT().Length().AnyTimes().Do(func() uint64 { var r uint64 = 0; return r })
		mockBc.EXPECT().Top().AnyTimes().Return(&blk)
		mockBc.EXPECT().Push(Any()).Do(func(block *block.Block) error {
			pushNumber = block.Head.Number
			return nil
		})

		p, _ := NewPoB(
			account.Account{
				ID:     "id0",
				Pubkey: []byte("pubkey"),
				Seckey: []byte("seckey"),
			},
			mockBc,
			mockPool,
			[]string{},
		)

		So(p.Account.GetId(), ShouldEqual, "id0")
		So(getNumber, ShouldEqual, 0)
		So(pushNumber, ShouldEqual, 0)
	})

}

func envinit(t *testing.T) (*PoB, []account.Account, []string, *txpool.TxPoolServer) {
	var accountList []account.Account
	var witnessList []string

	gopath := os.Getenv("GOPATH")
	blockDb1 := gopath + "/src/github.com/iost-official/Go-IOS-Protocol/consensus/pob/blockDB"
	txdb1 := gopath + "/src/github.com/iost-official/Go-IOS-Protocol/consensus/pob/txDB"
	blockDb2 := gopath + "/src/github.com/iost-official/blockDB"
	txdb2 := gopath + "/src/github.com/iost-official/txDB"

	delDir := os.RemoveAll(blockDb1)
	if delDir != nil {
		fmt.Println(delDir)
	}

	delDir = os.RemoveAll(txdb1)
	if delDir != nil {
		fmt.Println(delDir)
	}
	delDir = os.RemoveAll(blockDb2)
	if delDir != nil {
		fmt.Println(delDir)
	}
	delDir = os.RemoveAll(txdb2)
	if delDir != nil {
		fmt.Println(delDir)
	}
	if Exists(blockDb1) {
		fmt.Println(" Del blockDb1 Failed")
	}
	if Exists(blockDb2) {
		fmt.Println(" Del blockDb2 Failed")
	}
	if Exists(txdb1) {
		fmt.Println(" Del txdb1 Failed")
	}
	if Exists(txdb2) {
		fmt.Println(" Del txdb2 Failed")
	}

	acc := common.Base58Decode("3BZ3HWs2nWucCCvLp7FRFv1K7RR3fAjjEQccf9EJrTv4")
	_account, err := account.NewAccount(acc)
	if err != nil {
		panic("account.NewAccount error")
	}
	accountList = append(accountList, _account)
	witnessList = append(witnessList, _account.ID)
	//_accId := _account.ID

	for i := 1; i < 3; i++ {
		_account, err := account.NewAccount(nil)
		if err != nil {
			panic("account.NewAccount error")
		}
		accountList = append(accountList, _account)
		witnessList = append(witnessList, _account.ID)
	}

	tx.LdbPath = ""

	mockCtr := NewController(t)
	mockRouter := protocol_mock.NewMockRouter(mockCtr)

	network.Route = mockRouter
	guard := Patch(network.RouterFactory, func(_ string) (network.Router, error) {
		return mockRouter, nil
	})

	heightChan := make(chan message.Message, 1)
	blkSyncChan := make(chan message.Message, 1)
	mockRouter.EXPECT().FilteredChan(Any()).Return(heightChan, nil)
	mockRouter.EXPECT().FilteredChan(Any()).Return(blkSyncChan, nil)

	txChan := make(chan message.Message, 1)
	mockRouter.EXPECT().FilteredChan(Any()).Return(txChan, nil)

	blkChan := make(chan message.Message, 1)
	mockRouter.EXPECT().FilteredChan(Any()).Return(blkChan, nil)
	mockRouter.EXPECT().FilteredChan(Any()).Return(blkChan, nil).AnyTimes()

	defer guard.Unpatch()

	txDb := tx.TxDbInstance()
	if txDb == nil {
		panic("txDB error")
	}

	blockChain, err := block.Instance()
	if err != nil {
		panic("block.Instance error")
	}

	err = state.PoolInstance()
	if err != nil {
		panic("state.PoolInstance error")
	}

	sp, e := tx.NewServiPool(len(account.GenesisAccount), 1000)
	So(e, ShouldBeNil)

	tx.Data = tx.NewHolder(_account, state.StdPool, sp)

	blockChain.Top()
	p, err := NewPoB(accountList[0], blockChain, state.StdPool, witnessList)
	if err != nil {
		t.Errorf("NewPoB error")
	}
	fmt.Println("envinit ConfirmedLength:", p.blockCache.ConfirmedLength())
	blockCache := p.BlockCache()
	txPool, err := txpool.NewTxPoolServer(blockCache, blockCache.OnBlockChan())
	if err != nil {
		log.Log.E("NewTxPoolServer failed, stop the program! err:%v", err)
		os.Exit(1)
	}

	txPool.Start()
	return p, accountList, witnessList, txPool
}

func TestRunGenerateBlock(t *testing.T) {
	Convey("Test of Run (Generate Block)", t, func() {
		p, _, _, txpool := envinit(t)
		_tx := genTxMsg(p, 998)
		txpool.AddTransaction(&_tx)
		time.Sleep(time.Second * 1)

		bc := p.blockCache.LongestChain()
		pool := p.blockCache.LongestPool()
		blk := p.genBlock(p.account, bc, pool)
		So(len(blk.Content), ShouldEqual, 0)
		//So(len(blk.Content), ShouldEqual, 1)
		//So(blk.Content[0].Nonce, ShouldEqual, 998)
	})
}

func TestAddSinglesBlock(t *testing.T) {
	Convey("Test of Add singles block", t, func() {
		verify := func(blk *block.Block, parent *block.Block, pool state.Pool) (state.Pool, error) {
			return pool, nil
		}

		p, accountList, witnessList, _ := envinit(t)

		blockCnt := 10
		blockPool := genBlocks(p, accountList, witnessList, blockCnt, 0, true)

		var block *block.Block
		for i, blk := range blockPool {
			if i == 3 {
				So(p.blockCache.ConfirmedLength(), ShouldEqual, 2)
				block = blk
			} else {
				err := p.blockCache.Add(blk, verify)
				fmt.Println(err)
			}
		}
		So(p.blockCache.ConfirmedLength(), ShouldEqual, 2)

		err := p.blockCache.Add(block, verify)
		fmt.Println(err)

		So(p.blockCache.ConfirmedLength(), ShouldEqual, blockCnt-1)
	})
}

func TestRunConfirmBlock(t *testing.T) {
	Convey("Test of Run ConfirmBlock", t, func() {
		p, accList, witnessList, txpool := envinit(t)
		fmt.Println("0.TestRunConfirmBlock ConfirmedLength: ", p.blockCache.ConfirmedLength())
		_tx := genTxMsg(p, 998)
		txpool.AddTransaction(&_tx)
		fmt.Println("1.TestRunConfirmBlock ConfirmedLength: ", p.blockCache.ConfirmedLength())
		// block database delete failed
		initConfLength := p.blockCache.ConfirmedLength()

		for i := 0; i < 5; i++ {
			wit := ""
			for wit != witnessList[0] {
				currentTimestamp := consensus_common.GetCurrentTimestamp()
				wit = witnessOfTime(&p.globalStaticProperty, &p.globalDynamicProperty, currentTimestamp)
			}

			bc := p.blockCache.LongestChain()
			pool := p.blockCache.LongestPool()

			blk := p.genBlock(p.account, bc, pool)
			p.globalDynamicProperty.update(&blk.Head)
			err := p.blockCache.Add(blk, p.blockVerify)
			fmt.Println(err)
		}

		So(p.blockCache.ConfirmedLength(), ShouldEqual, initConfLength)
		for i := 1; i < 3; i++ {
			wit := ""
			for wit != witnessList[i] {
				currentTimestamp := consensus_common.GetCurrentTimestamp()
				wit = witnessOfTime(&p.globalStaticProperty, &p.globalDynamicProperty, currentTimestamp)
			}

			bc := p.blockCache.LongestChain()
			pool := p.blockCache.LongestPool()
			blk := p.genBlock(accList[i], bc, pool)
			p.globalDynamicProperty.update(&blk.Head)
			err := p.blockCache.Add(blk, p.blockVerify)
			fmt.Println(err)
			if i == 1 {
				So(p.blockCache.ConfirmedLength(), ShouldEqual, initConfLength)
			}
			if i == 2 {
				So(p.blockCache.ConfirmedLength(), ShouldEqual, initConfLength+5)
			}
		}
	})
}

func TestRunMultipleBlocks(t *testing.T) {
	Convey("Test of Run (Multiple Blocks)", t, func() {
		p, _, witnessList, txpool := envinit(t)
		_tx := genTxMsg(p, 998)
		txpool.AddTransaction(&_tx)

		bc := p.blockCache.LongestChain()
		pool := p.blockCache.LongestPool()

		for i := 100; i < 105; i++ {
			if i == 103 {
				bc = p.blockCache.LongestChain()
				pool = p.blockCache.LongestPool()
			}
			wit := ""
			for wit != witnessList[0] {

				currentTimestamp := consensus_common.GetCurrentTimestamp()
				wit = witnessOfTime(&p.globalStaticProperty, &p.globalDynamicProperty, currentTimestamp)
			}

			blk := p.genBlock(p.account, bc, pool)
			p.globalDynamicProperty.update(&blk.Head)

			blk.Head.Time = int64(i)

			headInfo := generateHeadInfo(blk.Head)
			sig, _ := common.Sign(common.Secp256k1, headInfo, p.account.Seckey)
			blk.Head.Signature = sig.Encode()

			err := p.blockCache.Add(blk, p.blockVerify)
			fmt.Println(err)
		}
	})
}

func generateTestBlockMsg(witness string, secKeyRaw string, number int64, parentHash []byte) (block.Block, message.Message) {
	blk := block.Block{
		Head: block.BlockHead{
			Number:     number,
			ParentHash: parentHash,
			Witness:    witness,
			Time:       consensus_common.GetCurrentTimestamp().Slot,
		},
	}
	headInfo := generateHeadInfo(blk.Head)
	sig, _ := common.Sign(common.Secp256k1, headInfo, common.Sha256([]byte(secKeyRaw)))
	blk.Head.Signature = sig.Encode()
	msg := message.Message{
		Time:    time.Now().Unix(),
		From:    "",
		To:      "",
		ReqType: int32(network.ReqNewBlock),
		Body:    blk.Encode(),
	}
	return blk, msg
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

//go test -bench=. -benchmem -run=nonce
func BenchmarkAddBlockCache(b *testing.B) {
	//benchAddBlockCache(b,10,true)
	benchAddBlockCache(b, 10, false)
}

func BenchmarkBlockVerifier(b *testing.B) { benchBlockVerifier(b) }
func BenchmarkTxCache(b *testing.B) {
	//benchTxCache(b,true)
	benchTxCache(b, true)
}

func BenchmarkBlockHead(b *testing.B) { benchBlockHead(b) }
func BenchmarkGenerateBlock(b *testing.B) {
	benchGenerateBlock(b, 6000)
}

func envInit(b *testing.B) (*PoB, []account.Account, []string, *txpool.TxPoolServer) {
	gopath := os.Getenv("GOPATH")
	blockDb1 := gopath + "/src/github.com/iost-official/Go-IOS-Protocol/consensus/pob/blockDB"
	txdb1 := gopath + "/src/github.com/iost-official/Go-IOS-Protocol/consensus/pob/txDB"
	blockDb2 := gopath + "/src/github.com/iost-official/blockDB"
	txdb2 := gopath + "/src/github.com/iost-official/txDB"

	delDir := os.RemoveAll(blockDb1)
	if delDir != nil {
		fmt.Println(delDir)
	}

	delDir = os.RemoveAll(txdb1)
	if delDir != nil {
		fmt.Println(delDir)
	}
	delDir = os.RemoveAll(blockDb2)
	if delDir != nil {
		fmt.Println(delDir)
	}
	delDir = os.RemoveAll(txdb2)
	if delDir != nil {
		fmt.Println(delDir)
	}
	if Exists(blockDb1) {
		fmt.Println(" Del blockDb1 Failed")
	}
	if Exists(blockDb2) {
		fmt.Println(" Del blockDb2 Failed")
	}
	if Exists(txdb1) {
		fmt.Println(" Del txdb1 Failed")
	}
	if Exists(txdb2) {
		fmt.Println(" Del txdb2 Failed")
	}

	var accountList []account.Account
	var witnessList []string

	acc := common.Base58Decode("3BZ3HWs2nWucCCvLp7FRFv1K7RR3fAjjEQccf9EJrTv4")
	_account, err := account.NewAccount(acc)
	if err != nil {
		panic("account.NewAccount error")
	}
	accountList = append(accountList, _account)
	witnessList = append(witnessList, _account.ID)
	_accId := _account.ID

	for i := 1; i < 3; i++ {
		_account, err := account.NewAccount(nil)
		if err != nil {
			panic("account.NewAccount error")
		}
		accountList = append(accountList, _account)
		witnessList = append(witnessList, _accId)
	}

	tx.LdbPath = ""

	mockCtr := NewController(b)
	mockRouter := protocol_mock.NewMockRouter(mockCtr)

	network.Route = mockRouter
	guard := Patch(network.RouterFactory, func(_ string) (network.Router, error) {
		return mockRouter, nil
	})

	heightChan := make(chan message.Message, 1)
	blkSyncChan := make(chan message.Message, 1)
	mockRouter.EXPECT().FilteredChan(Any()).Return(heightChan, nil)
	mockRouter.EXPECT().FilteredChan(Any()).Return(blkSyncChan, nil)

	txChan := make(chan message.Message, 1)
	mockRouter.EXPECT().FilteredChan(Any()).Return(txChan, nil)

	blkChan := make(chan message.Message, 1)
	mockRouter.EXPECT().FilteredChan(Any()).Return(blkChan, nil)
	mockRouter.EXPECT().FilteredChan(Any()).Return(blkChan, nil).AnyTimes()
	defer guard.Unpatch()

	txDb := tx.TxDbInstance()
	if txDb == nil {
		panic("txDB error")
	}

	blockChain, err := block.Instance()
	if err != nil {
		panic("block.Instance error")
	}

	err = state.PoolInstance()
	if err != nil {
		panic("state.PoolInstance error")
	}

	p, err := NewPoB(accountList[0], blockChain, state.StdPool, witnessList)
	if err != nil {
		b.Errorf("NewPoB error")
	}

	blockCache := p.BlockCache()
	txPool, err := txpool.NewTxPoolServer(blockCache, blockCache.OnBlockChan())
	if err != nil {
		log.Log.E("NewTxPoolServer failed, stop the program! err:%v", err)
		os.Exit(1)
	}

	txPool.Start()

	return p, accountList, witnessList, txPool
}

func genTx(p *PoB, nonce int) tx.Tx {
	main := lua.NewMethod(2, "main", 0, 1)
	code := `function main()
				return "success"
			end--f`
	lc := lua.NewContract(vm.ContractInfo{Prefix: "test", GasLimit: 10000, Price: 0, Publisher: vm.IOSTAccount(p.account.ID)}, code, main)

	_tx := tx.NewTx(int64(nonce), &lc)
	_tx, _ = tx.SignTx(_tx, p.account)
	return _tx
}

func genTxMsg(p *PoB, nonce int) message.Message {
	tx := genTx(p, nonce)

	txMsg := message.Message{
		Body:    tx.Encode(),
		ReqType: int32(network.ReqPublishTx),
	}
	return txMsg
}

func genBlockHead(p *PoB) {
	blk := block.Block{Content: []tx.Tx{}, Head: block.BlockHead{
		Version:    0,
		ParentHash: nil,
		Info:       []byte("test"),
		Number:     int64(1),
		Witness:    p.account.ID,
		Time:       int64(0),
	}}
	blk.Head.TreeHash = blk.CalculateTreeHash()
	headInfo := generateHeadInfo(blk.Head)
	sig, _ := common.Sign(common.Secp256k1, headInfo, p.account.Seckey)
	blk.Head.Signature = sig.Encode()
}

func genBlocks(p *PoB, accountList []account.Account, witnessList []string, n int, txCnt int, continuity bool) (blockPool []*block.Block) {
	confChain := p.blockCache.BlockChain()
	tblock := confChain.Top()

	slot := consensus_common.GetCurrentTimestamp().Slot

	for i := 0; i < n; i++ {
		var hash []byte
		if len(blockPool) == 0 {
			hash = tblock.HeadHash()
		} else {
			hash = blockPool[len(blockPool)-1].HeadHash()
		}
		//make every block has no parent
		if continuity == false {
			hash[i%len(hash)] = byte(i % 256)
		}
		blk := block.Block{Content: []tx.Tx{}, Head: block.BlockHead{
			Version:    0,
			ParentHash: hash,
			Info:       []byte("test"),
			Number:     int64(i + 1),
			Witness:    account.GetIdByPubkey(accountList[i%3].Pubkey),
			Time:       slot + int64(i),
		}}

		for i := 0; i < txCnt; i++ {
			blk.Content = append(blk.Content, genTx(p, i))
		}
		blk.Head.TreeHash = blk.CalculateTreeHash()
		headInfo := generateHeadInfo(blk.Head)
		sig, _ := common.Sign(common.Secp256k1, headInfo, accountList[i%3].Seckey)
		blk.Head.Signature = sig.Encode()
		blockPool = append(blockPool, &blk)
	}
	return
}
func benchAddBlockCache(b *testing.B, txCnt int, continuity bool) {

	p, accountList, witnessList, _ := envInit(b)
	blockPool := genBlocks(p, accountList, witnessList, b.N, txCnt, continuity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		p.blockCache.Add(blockPool[i], p.blockVerify)
		b.StopTimer()
	}

}

func benchGetBlock(b *testing.B, txCnt int, continuity bool) {
	p, accountList, witnessList, _ := envInit(b)
	blockPool := genBlocks(p, accountList, witnessList, b.N, txCnt, continuity)
	for i := 0; i < b.N; i++ {
		for _, bl := range blockPool {
			p.blockCache.Add(bl, p.blockVerify)
		}
	}

	//get block
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain := p.blockCache.LongestChain()
		b.StartTimer()
		chain.GetBlockByNumber(uint64(i))
		b.StopTimer()
	}
}

func benchBlockVerifier(b *testing.B) {
	p, accountList, witnessList, _ := envInit(b)
	blockPool := genBlocks(p, accountList, witnessList, 2, 6000, true)
	confChain := p.blockCache.BlockChain()
	tblock := confChain.Top()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.blockVerify(blockPool[0], tblock, state.StdPool)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func benchTxCache(b *testing.B, f bool) {
	p, _, _, txpool := envInit(b)
	var txs []message.Message

	for j := 0; j < b.N; j++ {
		_tx := genTxMsg(p, j)
		txpool.AddTransaction(&_tx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		txpool.AddTransaction(&txs[i])
		b.StopTimer()
	}

}

func benchTxCachePara(b *testing.B) {
	p, _, _, txpool := envInit(b)
	var txs []message.Message

	b.ResetTimer()
	for j := 0; j < 2000; j++ {
		_tx := genTxMsg(p, j)
		txs = append(txs, _tx)
		if j < 1000 {
			txpool.AddTransaction(&_tx)
		}
	}
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		start := time.Now().UnixNano()
		for j := 1000; j < 2000; j++ {
			txpool.AddTransaction(&txs[j])
		}
		end := time.Now().UnixNano()
		fmt.Println((end-start)/1000, " ns/op")
		wg.Done()
	}()
	wg.Wait()
}

func benchTxDb(b *testing.B, f bool) {
	p, _, _, txpool := envInit(b)
	var txs []tx.Tx
	txDb := tx.TxDbInstance()
	for j := 0; j < b.N; j++ {
		_tx := genTxMsg(p, j)
		txpool.AddTransaction(&_tx)
	}

	b.ResetTimer()
	if f == true {
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			txDb.Add(&txs[i])
			b.StopTimer()
		}
	} else {
		for i := 0; i < b.N; i++ {
			txDb.Add(&txs[i])
		}
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			txDb.Del(&txs[i])
			b.StopTimer()
		}

	}
}

func benchBlockHead(b *testing.B) {
	p, _, _, _ := envInit(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		genBlockHead(p)
		b.StopTimer()
	}
}

func benchGenerateBlock(b *testing.B, txCnt int) {
	p, _, _, txpool := envInit(b)
	TxPerBlk = txCnt

	for i := 0; i < TxPerBlk*b.N; i++ {
		_tx := genTxMsg(p, 998)
		txpool.AddTransaction(&_tx)
	}

	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		bc := p.blockCache.LongestChain()
		pool := p.blockCache.LongestPool()
		b.StartTimer()
		p.genBlock(p.account, bc, pool)
		b.StopTimer()
	}
}
