package tx

import (
	"errors"
	"fmt"
	"sync"

	"github.com/iost-official/Go-IOS-Protocol/common"
)

// Transaction Pool 实现内存Map存储
type TxPoolImpl struct {
	txMap map[string]*Tx
	lock  sync.RWMutex
}

// Tx Pool 初始化
func NewTxPoolImpl() *TxPoolImpl {
	return &TxPoolImpl{txMap: make(map[string]*Tx)}
}

// 在Tx Pool 插入一个 Tx
func (tp *TxPoolImpl) Add(tx *Tx) error {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	//fmt.Println("[tx_cache.Add]: ",tx.Nonce)
	tp.txMap[common.Base58Encode(tx.Hash())] = tx
	return nil
}

// 在Tx Pool 删除Tx
func (tp *TxPoolImpl) Del(tx *Tx) error {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	delete(tp.txMap, common.Base58Encode(tx.Hash()))
	return nil
}

// 在Tx Pool 获取Tx, 需要Tx的hash值
func (tp *TxPoolImpl) Get(hash []byte) (*Tx, error) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	tx, ok := tp.txMap[common.Base58Encode(hash)]
	if !ok {
		return nil, errors.New("Not Found")
	}
	return tx, nil
}

// 在Tx Pool 获取第一个Tx
func (tp *TxPoolImpl) Top() (*Tx, error) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	for _, tx := range tp.txMap {
		return tx, nil
	}
	return nil, errors.New("Empty")
}

// 判断一个Tx是否在Tx Pool
func (tp *TxPoolImpl) Has(tx *Tx) (bool, error) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	_, ok := tp.txMap[common.Base58Encode(tx.Hash())]
	return ok, nil
}

// 获取TxPool中tx的数量
func (tp *TxPoolImpl) Size() int {
	return len(tp.txMap)
}

// Transaction Pool 实现堆存储，维护Transaction的有序性
type TxPoolStack struct {
	txMap   map[string]int
	txStack []*Tx
	lock    sync.RWMutex
}

func NewTxPoolStack() (*TxPoolStack, error) {
	tp := TxPoolStack{
		txMap:   make(map[string]int),
		txStack: make([]*Tx, 1),
	}
	tp.txStack[0] = &Tx{}
	return &tp, nil
}

// 在Tx Pool 插入一个 Tx
func (tp *TxPoolStack) Add(tx *Tx) error {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.txStack = append(tp.txStack, nil)
	j := len(tp.txStack) - 1
	for j > 1 {
		if cmpTx(tp.txStack[j/2], tx) {
			tp.txStack[j] = tp.txStack[j/2]
			if _, ok := tp.txMap[common.Base58Encode(tp.txStack[j].Hash())]; ok {
				tp.txMap[common.Base58Encode(tp.txStack[j].Hash())] = j
			}
			j = j / 2
		} else {
			break
		}
	}
	tp.txStack[j] = tx
	tp.txMap[common.Base58Encode(tx.Hash())] = j
	return nil
}

// 在Tx Pool 删除Tx
func (tp *TxPoolStack) Del(tx *Tx) error {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	j := tp.txMap[common.Base58Encode(tx.Hash())]
	for j*2 < len(tp.txStack) {
		fmt.Println(j)
		nj := j * 2
		if (j*2+1 < len(tp.txStack)) && (cmpTx(tp.txStack[j*2+1], tp.txStack[j*2])) {
			nj = j*2 + 1
		}
		tp.txStack[j] = tp.txStack[nj]
		if _, ok := tp.txMap[common.Base58Encode(tp.txStack[j].Hash())]; ok {
			tp.txMap[common.Base58Encode(tp.txStack[j].Hash())] = j
		}
		j = nj
	}
	tp.txStack = tp.txStack[:len(tp.txStack)-1]
	delete(tp.txMap, common.Base58Encode(tx.Hash()))
	return nil
}

// 在Tx Pool 获取Tx, 需要Tx的hash值
func (tp *TxPoolStack) Get(hash []byte) (*Tx, error) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	j, ok := tp.txMap[common.Base58Encode(hash)]
	if !ok {
		return nil, errors.New("Not Found")
	}
	return tp.txStack[j], nil
}

// 在Tx Pool 获取第一个Tx
func (tp *TxPoolStack) Top() (*Tx, error) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	if len(tp.txStack) == 1 {
		return nil, errors.New("Empty")
	}
	return tp.txStack[1], nil
}

// 判断一个Tx是否在Tx Pool
func (tp *TxPoolStack) Has(tx *Tx) (bool, error) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	_, ok := tp.txMap[common.Base58Encode(tx.Hash())]
	return ok, nil
}

// 获取TxPool中tx的数量
func (tp *TxPoolStack) Size() int {
	return len(tp.txMap)
}

// 比较Tx中间的优先级，true即a>b，否则b>a
func cmpTx(a *Tx, b *Tx) bool {
	return true
}
