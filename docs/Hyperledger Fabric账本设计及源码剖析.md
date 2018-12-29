# Hyperledger Fabric账本设计及源码剖析

区块链技术的技术要点，简单来说有共识、P2P网络、密码学算法、智能合约及账本这几个方面。这篇文章我想简单阐述Hyperledger Fabric的账本设计，并简要分析源码。

账本简单的说，是一系列有序的、不可篡改的状态转移记录日志。状态转移是链码（chaincode）执行（交易）的结果，每个交易都是通过增删改操作提交一系列键值对到账本。一系列有序的交易被打包成块，这样就将账本串联成了区块链。同时，一个状态数据库维护账本当前的状态，因此也被叫做世界状态。现阶段，每个通道都有其账本，每个Peer节点都保存着其加入的通道的账本。

## 链
链实际上是交易日志，被结构化成哈希串起来的块，每个块包含N个有序交易。区块头信息里包含该区块所包含的所有交易的哈希（通过默克尔树实现），同时也包含前一个区块的哈希。这样，账本里的所有交易都被有序存储，并使用密码学强关联在一起。换句话说，账本的数据不可能被篡改，除非破坏哈希链。最新区块的哈希值实际上包含了从创始区块以来的所有交易，任何细微的改变都会使得当前区块的哈希变得不同。

## 状态数据库
账本当前状态数据实际上就是所有曾经在交易中出现的键值对的最新值。调用链码执行交易可以改变状态数据，为了高效的执行链码调用，所有数据的最新值都被存放在状态数据库中。就逻辑上来说，状态数据库仅仅是有序交易日志的快照，因此在任何时候都可以根据交易日志重新生成。状态数据库会在peer节点启动的时候自动恢复或重构，未完备前，该节点不会接受新的交易。

## 状态数据库的选择
状态数据库可以使用LevelDB或者CouchDB。LevelDB是默认的内置的数据库，CouchDB是额外的第三方数据库。跟LevelDB一样，CouchDB也能够存储任意的二进制数据，但是作为JSON文件存储数据库，CouchDB额外的支撑JSON富文本查询，如果链码的键值对存储的是JSON，那么可以很好的利用CouchDB的富文本查询功能。

两个数据库都支持链码的基本接口，设置键值对，获取键值对，键查询，区间键查询，组合键查询等等。

如果使用JSON结构化链码数据，可以使用CouchDB的JSON查询语言进行富文本查询，这样可以简单直观的理解账本里的数据。但是，虽然富文本查询对客户端应用程序来说是直观有效的，但是对Orderer不是很友好，不能保证结果的正确性，因为链码的模拟和区块提交是有时差的，中间的数据改变是富文本查询感知不到的。因此建议只在链码查询接口里使用富文本查询。

CouchDB是外置的第三方数据库，因此需要额外的管理成本，所以建议开始时使用内置LevelDB存储，在需要富文本查询时再切换成CouchDB。再者，最好使用JSON结构化链码数据结构，这样未来的扩展性会很好。

## 源码分析
### 接口定义

上面简要分析了Fabric的账本要点，为了更深入的理解，源代码分析必不可少。先看接口：

#### PeerLedgerProvider
``` go
// PeerLedgerProvider用于管理账本
type PeerLedgerProvider interface {
    // Create方法创建一个新账本，参数为创始块。创建账本&提交创世块这两个操作是原子的。
    // 创世块其实是一个配置信息块，包含Configure Transaction，具体可看configtxgen
    Create(genesisBlock *common.Block) (PeerLedger, error)
    // Open方法打开已存在的账本
    Open(ledgerID string) (PeerLedger, error)
    // Exists方法判断某个账本是否存在
    Exists(ledgerID string) (bool, error)
    // List方法列出当前节点的所有账本名字
    List() ([]string, error)
    // Close方法关闭该接口
   Close()
}
```
#### PeerLedger
``` go
// PeerLedger
type PeerLedger interface {
   commonledger.Ledger // 从0.6版本继承过来的接口，用于管理本地的区块
   // GetTransactionByID方法根据交易id获取交易详情
   GetTransactionByID(txID string) (*peer.ProcessedTransaction, error)
   // GetBlockByHash方法根据区块哈希获取区块详情
   GetBlockByHash(blockHash []byte) (*common.Block, error)
   // GetBlockByTxID方法根据交易ID获取区块详情
   GetBlockByTxID(txID string) (*common.Block, error)
   // GetTxValidationCodeByTxID方法根据交易ID获取交易验证编码
   GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error)
   // NewTxSimulator返回交易模拟器，客户端可以并行的获取多个交易模拟器
   NewTxSimulator() (TxSimulator, error)
   // NewQueryExecutor返回查询执行器，客户端可以并行的获取多个查询执行器
   NewQueryExecutor() (QueryExecutor, error)
   // NewHistoryQueryExecutor返回历史数据查询执行器，客户端可以并行的获取多个历史数据查询执行器
   NewHistoryQueryExecutor() (HistoryQueryExecutor, error)
   //Prune方法根据传递的削减策略削减区块/交易
   Prune(policy commonledger.PrunePolicy) error
}
```
#### QueryExecutor
``` go
// QueryExecutor用于执行查询
// Get*方法支持KV数据模型的查询. ExecuteQuery方法用于富文本查询
type QueryExecutor interface {
   // GetState方法查找特定namespace下某key的值，对链码来说，namespace就是chaincodeId
   GetState(namespace string, key string) ([]byte, error)
   // GetStateMultipleKeys方法查找特定namespace下一系列key的值
   GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error)
   // GetStateRangeScanIterator方法返回一个迭代器，其包含给定的key区间里所有的键值对（startKey包含在内，endKey排除在外）。空字符串的startKey表示第一个可用的key，空字符串的endKey表示最后一个可用的key
   GetStateRangeScanIterator(namespace string, startKey string, endKey string) (commonledger.ResultsIterator, error)
   // ExecuteQuery方法执行传递的查询语句并返回包含相关数据集的迭代器
   ExecuteQuery(namespace, query string) (commonledger.ResultsIterator, error)
   // Done方法释放资源
   Done()
}
```
#### TxSimulator
``` go
// TxSimulator模拟执行交易
// Set*相关方法执行KV数据模型的执行. ExecuteUpdate方法支持富文本操作
type TxSimulator interface {
   QueryExecutor // 交易模拟器包含查询执行器
   // SetState方法设置对应namespace下的key的当前value
   SetState(namespace string, key string, value []byte) error
   // DeleteState方法删除指定namespace下对应的key
   DeleteState(namespace string, key string) error
   // SetMultipleKeys方法设置对应namespace下多个key的value
   SetStateMultipleKeys(namespace string, kvs map[string][]byte) error
   // ExecuteUpdate方法支持富文本操作
   ExecuteUpdate(query string) error
   // GetTxSimulationResults方法包裹交易模拟结果
   GetTxSimulationResults() ([]byte, error)
}
```
#### HistoryQueryExecutor
``` go
// HistoryQueryExecutor执行历史查询
type HistoryQueryExecutor interface {
   // GetHistoryForKey方法返回key的历史数据迭代器
   GetHistoryForKey(namespace string, key string) (commonledger.ResultsIterator, error)
}
```
#### Ledger
``` go
// Ledger接口
type Ledger interface {
   // GetBlockchainInfo返回区块链基本信息
   GetBlockchainInfo() (*common.BlockchainInfo, error)
   // GetBlockByNumber根据区块编号返回区块信息
   GetBlockByNumber(blockNumber uint64) (*common.Block, error)
   // GetBlocksIterator返回区块迭代器，区块起始位置为startBlockNumber
   GetBlocksIterator(startBlockNumber uint64) (ResultsIterator, error)
   // Close方法关闭账本
   Close()
   // Commit方法提交新的区块到账本里
   Commit(block *common.Block) error
}
```
### 接口实现
至此，Fabric的账本接口基本介绍完毕，应该能大致看出其账本的设计思路。但还未涉及到具体的实现。接下来，我们深入到具体实现里，去看看基于LevelDB的状态数据库实现、历史数据库实现及本地账本的实现。

首先，深入kv_ledger_provider.go文件：
``` go
// Provider实现了接口ledger.PeerLedgerProvider
type Provider struct {
   idStore            *idStore
   blockStoreProvider blkstorage.BlockStoreProvider
   vdbProvider        statedb.VersionedDBProvider
   historydbProvider  historydb.HistoryDBProvider
}
type idStore struct {
   db *leveldbhelper.DB
}
```
可以看到，该Provider里包含了本地账本Provider，状态数据库Provider，历史数据库Provider以及idStore。idStore内部封装了LevelDB的接口，存储的是多通道的标识。KVLedgerProvider的层次还是在多链多通道的高度，剥离ledgerID，我们进入单链结构。一个链由账本数据库、状态数据库及历史数据库组成。

#### 历史数据库
``` go
// HistoryDBProvider provides an instance of a history DB
type HistoryDBProvider interface {
   // GetDBHandle returns a handle to a HistoryDB
   GetDBHandle(id string) (HistoryDB, error)
   // Close closes all the HistoryDB instances and releases any resources held by HistoryDBProvider
   Close()
}
// HistoryDBProvider implements interface HistoryDBProvider
type HistoryDBProvider struct {
   dbProvider *leveldbhelper.Provider
}
```
HistoryDBProvider实现了HistoryDBProvider接口，内部封装了一个LevelDB对象，可根据dbName返回不同的历史数据库。
``` go
// HistoryDB - an interface that a history database should implement
type HistoryDB interface {
   NewHistoryQueryExecutor(blockStore blkstorage.BlockStore) (ledger.HistoryQueryExecutor, error)
   Commit(block *common.Block) error
   GetLastSavepoint() (*version.Height, error)
   ShouldRecover(lastAvailableBlock uint64) (bool, uint64, error)
   CommitLostBlock(block *common.Block) error
}
// historyDB implements HistoryDB interface
type historyDB struct {
   db     *leveldbhelper.DBHandle
   dbName string
}
```
HistoryDB接口定义了历史数据库可以进行的操作，而内部类historyDB实现了该接口。

GetLastSavepoint方法获取历史数据库最后一次更新的区块编号及交易索引（封装在Height类中），LastSavepoint的记录实现方法就是在commit区块的时候，更新savePointKey键值对，然后由GetLastSavepoint方法读取该键值对，得到最后提交点数据。

ShouldRecover方法判断是否需要重构历史数据库，返回值为1:、是否需要；2、需要从哪个块开始重构；3、错误消息。实现方法很简单，就是将参数lastAvailableBlock跟GetLastSavepoint方法返回的区块编号进行对比，如果不一样就需要重构，重构点为savepoint的区块编号+1。

CommitLostBlock方法是Commit方法的一层封装。下面我们详细剖析Commit方法的实现：
``` go
// Commit implements method in HistoryDB interface
func (historyDB *historyDB) Commit(block *common.Block) error {
   blockNo := block.Header.Number  // 当前处理区块的编号
   var tranNo uint64 // 初始化交易编号为0，代表区块xx的第0个交易

   dbBatch := leveldbhelper.NewUpdateBatch() // 数据库批量更新事物
   txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
   if len(txsFilter) == 0 {
      txsFilter = util.NewTxValidationFlags(len(block.Data.Data))
      block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
   } // 获取区块交易的状态过滤，（区块中的交易不是所有都是有效的，但历史数据库中只记录有效交易）

   // 循环读取交易
   for _, envBytes := range block.Data.Data {

      // If the tran is marked as invalid, skip it
      if txsFilter.IsInvalid(int(tranNo)) {
         tranNo++
         continue
      } // 如果该交易是无效的，略过，并自增交易编号

      env, err := putils.GetEnvelopeFromBlock(envBytes)
      if err != nil {
         return err
      } // 反序列化交易信封

      payload, err := putils.GetPayload(env)
      if err != nil {
         return err
      } // 反序列化交易负载

      chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
      if err != nil {
         return err
      } // 反序列化交易头

      // 如果不是endorser交易，则略过
            if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {

               respPayload, err := putils.GetActionFromEnvelope(envBytes)
               if err != nil {
                  return err
               } // 反序列化交易背书结果

               txRWSet := &rwsetutil.TxRwSet{}
               if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
                  return err
               } // 从背书交易结果中反序列化交易读写集
               // 循环交易读写集
               for _, nsRWSet := range txRWSet.NsRwSets {
                  ns := nsRWSet.NameSpace // 链码id

                  // 循环交易读写集的写集
                  for _, kvWrite := range nsRWSet.KvRwSet.Writes {
                     writeKey := kvWrite.Key

                     // 将ns、写key、区块编号、交易编号合并成一个组合key
                     compositeHistoryKey := historydb.ConstructCompositeHistoryKey(ns, writeKey, blockNo, tranNo)

                     // 将组合key写入事物
                     dbBatch.Put(compositeHistoryKey, emptyValue)
                  }
               }
            }
            tranNo++ // 处理完一个交易，自增交易编号
         }

         // 处理完一个区块，就更新savepoint
         height := version.NewHeight(blockNo, tranNo)
         dbBatch.Put(savePointKey, height.ToBytes())

         // 向db写数据
            if err := historyDB.db.WriteBatch(dbBatch, false); err != nil {
               return err
            }
            return nil
         }
```
需要注意的是，历史数据库并不存储key的值，而只存储在某个区块的某个交易里，某key变动了一次。后续需要查询的时候，根据变动历史去查询实际变动的值，这就是HistoryQueryExecutor的作用。NewHistoryQueryExecutor方法接受一个实际存储数据的账本作为参数，返回HistoryQueryExecutor。
``` go
// LevelHistoryDBQueryExecutor is a query executor against the LevelDB history DB
type LevelHistoryDBQueryExecutor struct {
   historyDB  *historyDB
   blockStore blkstorage.BlockStore
}

// GetHistoryForKey implements method in interface `ledger.HistoryQueryExecutor`
func (q *LevelHistoryDBQueryExecutor) GetHistoryForKey(namespace string, key string) (commonledger.ResultsIterator, error) {

   if ledgerconfig.IsHistoryDBEnabled() == false {
      return nil, errors.New("History tracking not enabled - historyDatabase is false")
   }

   var compositeStartKey []byte
   var compositeEndKey []byte
   compositeStartKey = historydb.ConstructPartialCompositeHistoryKey(namespace, key, false)
   compositeEndKey = historydb.ConstructPartialCompositeHistoryKey(namespace, key, true)

   // range scan to find any history records starting with namespace~key
   dbItr := q.historyDB.db.GetIterator(compositeStartKey, compositeEndKey)
   return newHistoryScanner(compositeStartKey, namespace, key, dbItr, q.blockStore), nil
}
//historyScanner implements ResultsIterator for iterating through history results
type historyScanner struct {
   compositePartialKey []byte //compositePartialKey includes namespace~key
   namespace           string
   key                 string
   dbItr               iterator.Iterator
   blockStore          blkstorage.BlockStore
}
```
GetHistoryForKey方法将传递过来的key组合后，生成一个startKey和一个endKey，LevelDB可以进行key的模糊查询，基于此功能，从historyDB的LevelDB接口中获取key区间的迭代器，然后生成historyScanner，该类实现了ResultsIterator接口，基于此接口，就可以进行key的历史数据查询了。
``` go
func (scanner *historyScanner) Next() (commonledger.QueryResult, error) {
   if !scanner.dbItr.Next() {
      return nil, nil
   } // LevelDB的迭代器没数据了，直接就返回

   historyKey := scanner.dbItr.Key() // ns+key+区块编号+交易编号

   _, blockNumTranNumBytes := historydb.SplitCompositeHistoryKey(historyKey, scanner.compositePartialKey) // 分割出区块编号+交易编号
   blockNum, bytesConsumed := util.DecodeOrderPreservingVarUint64(blockNumTranNumBytes[0:])
   tranNum, _ := util.DecodeOrderPreservingVarUint64(blockNumTranNumBytes[bytesConsumed:])
   // 获取区块编号、交易编号（二进制分割）

   tranEnvelope, err := scanner.blockStore.RetrieveTxByBlockNumTranNum(blockNum, tranNum)
   if err != nil {
      return nil, err
   } // 获取跟当前key相关的交易信息

   queryResult, err := getKeyModificationFromTran(tranEnvelope, scanner.namespace, scanner.key)
   if err != nil {
      return nil, err
   } // 获取当前查询结果
   return queryResult, nil
}

// getTxIDandKeyWriteValueFromTran inspects a transaction for writes to a given key
func getKeyModificationFromTran(tranEnvelope *common.Envelope, namespace string, key string) (commonledger.QueryResult, error) {
   payload, err := putils.GetPayload(tranEnvelope)
   if err != nil {
      return nil, err
   }
   tx, err := putils.GetTransaction(payload.Data)
   if err != nil {
      return nil, err
   }
   _, respPayload, err := putils.GetPayloads(tx.Actions[0])
   if err != nil {
      return nil, err
   }
   chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
   if err != nil {
      return nil, err
   }
   txID := chdr.TxId
   timestamp := chdr.Timestamp
   txRWSet := &rwsetutil.TxRwSet{}
   if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
      return nil, err
   }
   // 在之前的分析中已存在

   // 在交易读写集中查询该key，得到就返回查询结果，如果没有，就报错
   for _, nsRWSet := range txRWSet.NsRwSets {
      if nsRWSet.NameSpace == namespace {
         for _, kvWrite := range nsRWSet.KvRwSet.Writes {
            if kvWrite.Key == key {
               return &queryresult.KeyModification{TxId: txID, Value: kvWrite.Value,
                  Timestamp: timestamp, IsDelete: kvWrite.IsDelete}, nil
            }
         }
         return nil, errors.New("Key not found in namespace's writeset")
      }
   }
   return nil, errors.New("Namespace not found in transaction's ReadWriteSets")
}
```
至此，我们应当对历史数据库的实现有了一定的认识。下面介绍状态数据库：

#### 状态数据库
状态数据库的实现现在有两种方式，LevelDB以及CouchDB。这里暂时不剖析CouchDB实现的状态数据库。
``` go
// VersionedDBProvider接口定义如何管理VersionedDB
type VersionedDBProvider interface {
   GetDBHandle(id string) (VersionedDB, error)
   Close()
}

// VersionedDB接口定义状态数据库应该有哪些操作
type VersionedDB interface {
   // GetState方法返回key当前值
   GetState(namespace string, key string) (*VersionedValue, error)
   // GetStateMultipleKeys方法多个key的当前值
   GetStateMultipleKeys(namespace string, keys []string) ([]*VersionedValue, error)
   // GetStateRangeScanIterator方法返回key区间的键值对迭代器
   GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ResultsIterator, error)
   // ExecuteQuery方法执行富文本查询，LevelDB版本未实现该方法，略过
   ExecuteQuery(namespace, query string) (ResultsIterator, error)
   // ApplyUpdates方法更新数据库
   ApplyUpdates(batch *UpdateBatch, height *version.Height) error
   // GetLatestSavePoint方法返回最后一次更新的区块交易编号
   GetLatestSavePoint() (*version.Height, error)
   // ValidateKey方法校验key，LevelDB略过
   ValidateKey(key string) error
   // Open opens the db
   Open() error
   // Close closes the db
   Close()
}
```
在阅读源码时还需注意，状态数据库的kv不仅仅是string，他们是特定类型的序列化：
``` go
// CompositeKey包括ns和key
type CompositeKey struct {
   Namespace string
   Key       string
}
// VersionedValue包含值和当前的version，version跟区块编号相关
type VersionedValue struct {
   Value   []byte
   Version *version.Height
}

// VersionedKV 联合 key 和 value
type VersionedKV struct {
   CompositeKey
   VersionedValue
}
```
下面看LevelDB版本的StateDB，代码里叫VersionedDB。
``` go
// GetState implements method in VersionedDB interface
func (vdb *versionedDB) GetState(namespace string, key string) (*statedb.VersionedValue, error) {
   compositeKey := constructCompositeKey(namespace, key) // 将ns和key组合成一个key
   dbVal, err := vdb.db.Get(compositeKey)
   if err != nil {
      return nil, err
   } // 通过LevelDB接口获取key的值

   if dbVal == nil {
      return nil, nil
   }

   // 解析数据库中的value，封装成VersionedValue
   val, ver := statedb.DecodeValue(dbVal)
   return &statedb.VersionedValue{Value: val, Version: ver}, nil
}
```
GetStateMultipleKeys方法即循环调用GetState，无需详解

GetStateRangeScanIterator方法利用LevelDB的key模糊查询，返回LevelDB迭代器。再使用kvScanner封装读操作。
``` go
// ApplyUpdates implements method in VersionedDB interface
func (vdb *versionedDB) ApplyUpdates(batch *statedb.UpdateBatch, height *version.Height) error {
   dbBatch := leveldbhelper.NewUpdateBatch()  // 数据库事物
   namespaces := batch.GetUpdatedNamespaces() // 更新batch的ns列表
   for _, ns := range namespaces { // 循环ns列表
      updates := batch.GetUpdates(ns)
      for k, vv := range updates { // 循环ns的更新列表
         compositeKey := constructCompositeKey(ns, k)

         if vv.Value == nil {
            dbBatch.Delete(compositeKey)
         } else {
            dbBatch.Put(compositeKey, statedb.EncodeValue(vv.Value, vv.Version))
         }
         // 根据更新，将对应操作插入数据库事物中
      }
   }
   dbBatch.Put(savePointKey, height.ToBytes())
   if err := vdb.db.WriteBatch(dbBatch, false); err != nil {
      return err
   } // 更新savepoint
   return nil
}
```
相较历史状态数据库，状态数据库更简单，它不用关心区块或者交易，仅仅只关心key的最新值就可以了。

### 账本数据库
Fabric的账本是基于文件系统，将区块存储于文件块中，在LevelDB中存储区块交易对应的文件块及其偏移。账本数据库同样有Provider和相应的DB实现，翻阅代码发现实际上主体是blockfileMgr这个类实现相关的业务逻辑。该管理器有以下管理功能：
1. 管理文件存储路径
2. 管理存储区块的独立文件
3. 追踪最新文件的 checkpoint
4. 管理区块和交易的索引
当管理器启动的时候，它会去检测当前是第一次启动还是重启。区块文件在文件系统中以顺序编号存储，同时每个文件块都有固定的大小，例如：blockfile_000000，blockfile_000001等。

每个交易在存储的时候都记录了该交易的大小：
Adding txLoc [fileSuffixNum=0, offset=3, bytesLength=104] for tx [1:0] to index
Adding txLoc [fileSuffixNum=0, offset=107, bytesLength=104] for tx [1:1] to index
区块在存储的时候，不仅存储区块交易的信息，同时存储交易位置偏移量信息。

在该manager启动的时候，会执行以下操作：

1. 检查存储文件的路径是否存在，不存在则创建
2. 检查索引数据库是否存在，不存在则创建
3. 设置checkpoint：
  * 从数据库读取cpinfo，如果不存在，创建新的cpinfo
  * 如果cpinfo是从数据库中读取的，则开始和文件系统比较
  * 如果cpinfo和文件系统不同步，那么从文件系统重新生成cpinfo
4. 启动新的文件writer，在checkpoint截断文件
5. 处理区块和交易的索引信息
  * 实例化新的blockIdxInfo
  * 如果数据库中存在索引了，载入blockIdxInfo
  * 同步索引
6. 通过APIs更新区块链信息

以上6点就是初始化mananger的过程。接下来，我们分拆fsblkstorage的底层支撑类。

首先看区块是如何序列化的。
``` go
// 序列化block类，包含三个字段，区块头信息、区块元信息以及区块里包含的交易信息
type serializedBlockInfo struct {
   blockHeader *common.BlockHeader
   txOffsets   []*txindexInfo
   metadata    *common.BlockMetadata
}

//交易索引类标识了该交易在当前区块的偏移量以及交易数据长度
type txindexInfo struct {
   txID string
   loc  *locPointer
}

// 该方法揭示了如何序列化区块。简单的说就是将区块头、交易数据、元数据以顺序的方式通过protobuffer的序列化后拼接在一起，形成byte数组，二进制数据流。随后就可将该数据流通过文件writer写入文件块
func serializeBlock(block *common.Block) ([]byte, *serializedBlockInfo, error) {
   buf := proto.NewBuffer(nil)
   var err error
   info := &serializedBlockInfo{}
   info.blockHeader = block.Header
   info.metadata = block.Metadata
   if err = addHeaderBytes(block.Header, buf); err != nil {
      return nil, nil, err
   }
   if info.txOffsets, err = addDataBytes(block.Data, buf); err != nil {
      return nil, nil, err
   }
   if err = addMetadataBytes(block.Metadata, buf); err != nil {
      return nil, nil, err
   }
   return buf.Bytes(), info, nil
}
 // 该类同时还有些序列化，反序列化方法。基本是Protobuffer的序列化方法的封装，不深究
上面揭示了区块是如何序列化以及反序列化的。下面介绍文件块是如何读取的：

// blockfileStream类从单个文件读取区块
type blockfileStream struct {
   fileNum       int // 文件块编号（000000,000001）
   file          *os.File // 文件对象
   reader        *bufio.Reader // 缓存读对象
   currentOffset int64 // 当前文件读取偏移量
}


// blockStream类读取多个文件块
type blockStream struct {
   rootDir           string // 文件块的目录
   currentFileNum    int // 当前文件块编号
   endFileNum        int // 结束文件块编号
   currentFileStream *blockfileStream // 封装的当前文件块的读取流对象
}


// blockPlacementInfo类封装了区块在文件块中的位置信息
type blockPlacementInfo struct {
   fileNum          int // 文件块编号
   blockStartOffset int64 // 区块在文件块的起始偏移量
   blockBytesOffset int64 // 区块的长度
}
```
blockfileStream 和 blockStream 都有 nextBlockBytes、nextBlockBytesAndPlacementInfo和close方法。分别是获取下一个区块的数据流、和区块相对位置以及关闭文件块流。
同时，封装了文件块的writer和reader，方法基本同标准库保持一致。

接下来，剖析账本数据库的索引系统，文件模式的区块存储方式如果没有快速定位的索引信息，那么查询区块交易信息可能是噩梦。Fabric使用LevelDB作为文件块的索引实现方式。下面详解：

现阶段支持的索引有：

* 区块编号
* 区块哈希
* 交易ID索引交易
* 区块交易编号
* 交易ID索引区块
* 交易ID索引交易验证码

同时，Fabric定义了index接口，可以定制索引的实现：
``` go
type index interface {
   getLastBlockIndexed() (uint64, error)  // 获取最后被索引的区块编号
   indexBlock(blockIdxInfo *blockIdxInfo) error // 创建区块索引
   getBlockLocByHash(blockHash []byte) (*fileLocPointer, error) // 通过区块哈希获取区块位置
   getBlockLocByBlockNum(blockNum uint64) (*fileLocPointer, error) // 通过区块编号获取区块位置
   getTxLoc(txID string) (*fileLocPointer, error) // 通过交易ID获取交易位置
   getTXLocByBlockNumTranNum(blockNum uint64, tranNum uint64) (*fileLocPointer, error) // 通过区块交易编号获取交易位置
   getBlockLocByTxID(txID string) (*fileLocPointer, error) // 通过交易ID获取区块位置
   getTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) // 通过交易ID获取交易验证码
}
```
getLastBlockIndexed方法的实现就是在LevelDB中存储了个键值对，每次创建区块索引后更新该键值对；

indexBlock方法是创建索引的唯一方法，它接收blockIdxInfo类为参数，依次生成6个索引，方式就是组合查询键，值为所在文件偏移位置。交易偏移是区块偏移和交易相对偏移的组合。

剩下的几个方法就是读取索引了，略过。

最后还剩下一个迭代器，其实现就是整合上面介绍类的整合应用。

## 总结
以上就简单介绍了Hyperledger Fabric的账本设计相关的内容，应该会让大家理解其账本设计更加容易。
