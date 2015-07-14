// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package block

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
    fct "github.com/FactomProject/factoid"
)

type IFBlock interface {
	fct.IBlock
	// Get the ChainID. This is a constant for all Factoids.
	GetChainID() fct.IHash
	// Validation functions
	Validate() (bool, error)
    ValidateTransaction(fct.ITransaction) bool
    // Marshal just the transactions.  This is because we need the length
	MarshalTrans() ([]byte, error)
    // Add a coinbase transaction.  This transaction has no inputs
    AddCoinbase(fct.ITransaction) (bool, error)
    // Add a proper transaction.  Transactions are validated before
    // being added to the block.
	AddTransaction(fct.ITransaction) (bool, error)
    // Calculate all the MR and serial hashes for this block.  Done just
    // prior to being persisted.
	CalculateHashes()
    // Hash accessors
    GetBodyMR() fct.IHash
    GetPrevKeyMR() fct.IHash
    SetPrevKeyMR([]byte) 
    GetPrevFullHash() fct.IHash
    SetPrevFullHash([]byte) 
    // Accessors for the Directory Block Height
    SetDBHeight(uint32)
	GetDBHeight() uint32
	// Accessors for the Exchange rate
	SetExchRate(uint64)
	GetExchRate() uint64
    // Accessors for the transactions
    GetTransactions() []fct.ITransaction
    
    // Mark an end of Minute.  If there are multiple calls with the same minute value
    // the later one simply overwrites the previous one.  Since this is an informational 
    // data point, we do not enforce much, other than order (the end of period one can't
    // come before period 2.  We just adjust the periods accordingly.
    EndOfPeriod(min int)
}

// FBlockHeader defines information about a block and is used in the bitcoin
// block (MsgBlock) and headers (MsgHeaders) messages.
//
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#factoid-block
//
type FBlock struct {
	IFBlock
	//  ChainID         IHash     // ChainID.  But since this is a constant, we need not actually use space to store it.
	BodyMR              fct.IHash // Merkle root of the Factoid transactions which accompany this block.
	PrevKeyMR           fct.IHash // Key Merkle root of previous block.
	PrevFullHash        fct.IHash // Sha3 of the previous Factoid Block
	ExchRate            uint64    // Factoshis per Entry Credit
	DBHeight            uint32    // Directory Block height
    // Header Expansion Size  varint
	// Transaction count
	// body size
	transactions []fct.ITransaction // List of transactions in this block
	
	endOfPeriod [10]int  // End of Minute transaction heights.  The mark the height of the first entry of 
                         // the NEXT period.  This entry may not exist.  The Coinbase transaction is considered
                         // to be in the first period.  Factom's periods will initially be a minute long, and
                         // there will be 10 of them.  This may change in the future.
}

var _ IFBlock = (*FBlock)(nil)

func (b *FBlock) EndOfPeriod(period int) {
    period = period-1                            // Make the period zero based.  
    if period < 0 || period >= 10 { return }     // Ignore out of range period.
    for i := period; i < 10; i++ {               // Set the period and all following to the height
        b.endOfPeriod[i] = len(b.transactions)
    }
}


func (b *FBlock) GetTransactions() []fct.ITransaction {
    return b.transactions
}

func (b FBlock) GetNewInstance() fct.IBlock {
    return new(FBlock)
}

func (FBlock) GetDBHash() fct.IHash {
    return fct.Sha([]byte("FBlock"))
}

func (b *FBlock) GetHash() fct.IHash {
    if b.BodyMR != nil && bytes.Compare(b.BodyMR.Bytes(), new([32]byte)[:])!=0 {
        return b.BodyMR
    }
    b.CalculateHashes()
    return b.BodyMR
}

func (b *FBlock) MarshalTrans() ([]byte, error) {
	var out bytes.Buffer
	var periodMark = 0;
	for i, trans := range b.transactions {
        
        for periodMark < len(b.endOfPeriod) && i == b.endOfPeriod[periodMark] {
            out.WriteByte(0xFF)
            periodMark++
        }
        
		data, err := trans.MarshalBinary()
		if err != nil {
			return nil, err
		}
		out.Write(data)
		if err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

// Write out the block
func (b *FBlock) MarshalBinary() ([]byte, error) {
	var out bytes.Buffer
	
	out.Write(fct.FACTOID_CHAINID)
    
    if b.BodyMR == nil {b.BodyMR = new(fct.Hash)}
    data, err := b.GetHash().MarshalBinary()
	if err != nil {
		return nil, err
	}
	out.Write(data)
    
    if b.PrevKeyMR == nil {b.PrevKeyMR = new(fct.Hash)}
	data, err = b.PrevKeyMR.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out.Write(data)
    
    if b.PrevFullHash == nil {b.PrevFullHash = new(fct.Hash)}
    data, err = b.PrevFullHash.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out.Write(data)
    
	binary.Write(&out, binary.BigEndian, uint64(b.ExchRate))
	binary.Write(&out, binary.BigEndian, uint32(b.DBHeight))
    
    fct.EncodeVarInt(&out,0)        // At this point in time, nothing in the Expansion Header
                                    // so we just write out a zero.
    
	binary.Write(&out, binary.BigEndian, uint32(len(b.transactions)))

	transdata, err := b.MarshalTrans()                           // first get trans data
	binary.Write(&out, binary.BigEndian, uint32(len(transdata))) // write out its length
	out.Write(transdata)                                         // write out trans data

	return out.Bytes(), nil
}

// UnmarshalBinary assumes that the Binary is all good.  We do error
// out if there isn't enough data, or the transaction is too large.
func (b *FBlock) UnmarshalBinaryData(data []byte) ([]byte, error) {
	if bytes.Compare(data[:fct.ADDRESS_LENGTH], fct.FACTOID_CHAINID[:]) != 0 {
		return nil, fmt.Errorf("Block does not begin with the Factoid ChainID")
	}
	data = data[32:]
	
	b.BodyMR = new(fct.Hash)
	data, err := b.BodyMR.UnmarshalBinaryData(data)
	if err != nil {
		return nil, err
	}

	b.PrevKeyMR = new(fct.Hash)
	data, err = b.PrevKeyMR.UnmarshalBinaryData(data)
	if err != nil {
		return nil, err
	}
	
	b.PrevFullHash = new(fct.Hash)
	data, err = b.PrevFullHash.UnmarshalBinaryData(data)
	if err != nil {
		return nil, err
	}

	b.ExchRate, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	b.DBHeight, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	skip, data := fct.DecodeVarInt(data) // Skip the Expansion Header, if any, since
    data = data[skip:]                   // we don't know what to do with it.

	cnt, data := binary.BigEndian.Uint32(data[0:4]), data[4:]

	data = data[4:] // Just skip the size... We don't really need it.

	b.transactions = make([]fct.ITransaction, cnt, cnt)
    var periodMark = 1
	for i := uint32(0); i < cnt; i++ {
        
        for data[0] == 0xFF {
            b.EndOfPeriod(periodMark)
            data = data[1:]
            periodMark++
        }
        
        trans := new(fct.Transaction)
        data,err = trans.UnmarshalBinaryData(data)
        if err != nil {
            return nil, fmt.Errorf("Failed to unmarshal a transaction in block.\n%s",b.String())
        }
		b.transactions[i] = trans
	}
	return data, nil
}

func (b *FBlock) UnmarshalBinary(data []byte) (err error) {
	data, err = b.UnmarshalBinaryData(data)
	return err
}

// Tests if the transaction is equal in all of its structures, and
// in order of the structures.  Largely used to test and debug, but
// generally useful.
func (b1 *FBlock) IsEqual(block fct.IBlock) []fct.IBlock {

    if(b1.endOfPeriod[9]==0){
        b1.EndOfPeriod(1)           // Sets the end of the first period here.
    }                               // This is what unmarshalling will do.
    
	b2, ok := block.(*FBlock)

	if !ok || // Not the right kind of IBlock
        b1.ExchRate != b2.ExchRate ||
        b1.DBHeight != b2.DBHeight {
            r := make([]fct.IBlock,0,3)
            return append(r,b1)
        }
        
    r := b1.BodyMR.IsEqual(b2.BodyMR)
    if r != nil {
        return append(r,b1)
    }
    r = b1.PrevKeyMR.IsEqual(b2.PrevKeyMR)
    if r != nil {
        return append(r,b1)
    }
    r = b1.PrevFullHash.IsEqual(b2.PrevFullHash) 
    if r != nil {
        return append(r,b1)
    }

    if b1.endOfPeriod != b2.endOfPeriod {
        fmt.Println(b1.endOfPeriod)
        fmt.Println(b2.endOfPeriod)
        return append(r,b1)
    }

	for i, trans := range b1.transactions {
		r := trans.IsEqual(b2.transactions[i]) 
		if r != nil {
            return append(r,b1)
		}
	}

	return nil
}
func (b *FBlock) GetChainID() fct.IHash {
    h := new(fct.Hash)
    h.SetBytes(fct.FACTOID_CHAINID)
    return h
}
func (b *FBlock) GetBodyMR() fct.IHash {
    return b.BodyMR
}
func (b *FBlock) GetPrevKeyMR() fct.IHash {
    return b.PrevKeyMR
}
func (b *FBlock) SetPrevKeyMR(hash []byte) {
    h := fct.NewHash(hash)
    b.PrevKeyMR= h
}
func (b *FBlock) GetPrevFullHash() fct.IHash {
    return b.PrevFullHash
}
func (b *FBlock) SetPrevFullHash(hash[]byte)  {
    b.PrevFullHash.SetBytes(hash)
}

func (b *FBlock) CalculateHashes() {
    hashes := make([]fct.IHash,0,len(b.transactions))
    for _,trans := range b.transactions{
        hashes = append(hashes,trans.GetHash())
    }
    b.BodyMR = fct.ComputeMerkleRoot(hashes)
}

func (b *FBlock) SetDBHeight(dbheight uint32) {
	b.DBHeight = dbheight
}
func (b *FBlock) GetDBHeight() uint32 {
	return b.DBHeight
}
func (b *FBlock) SetExchRate(rate uint64) {
    b.ExchRate = rate
}
func (b *FBlock) GetExchRate() uint64 {
	return b.ExchRate
}

func (b FBlock) ValidateTransaction(trans fct.ITransaction) bool {
    // Calculate the fee due.
    {
        valid := trans.Validate()
        if valid != fct.WELL_FORMED {
            return false // Transaction is not well formed.
        }
    }
    if len(b.transactions)>0{
        ok := trans.ValidateSignatures()
        if !ok {
            return false // Transaction is not properly signed.
        }
    }
    fee, err := trans.CalculateFee(b.ExchRate)
    if err != nil {
        return false
    }
    tin,ok1  := trans.TotalInputs()
    tout,ok2 := trans.TotalOutputs()
    tec,ok3  := trans.TotalECs()
    sum,ok4  := fct.ValidateAmounts(tout,tec,fee)
    if !ok1 || !ok2 || !ok3 || !ok4 || tin < sum {
        return false    // Transaction either has not enough inputs, 
    }                   // or didn't pay the fee.
    return true
}

func (b FBlock) Validate() (bool, error) {
	for i, trans := range b.transactions {
        if !b.ValidateTransaction(trans) {
            return false, fmt.Errorf("Block contains invalid transactions")
        }
        if i == 0 {
            if len(trans.GetInputs()) != 0 {
                return false, fmt.Errorf("Block has a coinbase transaction with inputs")
            }
        }else{
            if len(trans.GetInputs()) == 0 {
                return false, fmt.Errorf("Block contains transactions without inputs")
            }
        }
	}

	// Need to check balances are all good.

	// Save what we got for our hashes
	mr := b.BodyMR
	pb := b.PrevKeyMR
	ph := b.PrevFullHash

	// Recalculate the hashes
	b.CalculateHashes()

	// Make sure nothing changes.  If something did, this block is bad.
	return mr == b.BodyMR && pb == b.PrevKeyMR && ph == b.PrevFullHash, nil
}

// Add the first transaction of a block.  This transaction makes the 
// payout to the servers, so it has no inputs.   This transaction must
// be deterministic so that all servers will know and expect its output.
func (b *FBlock) AddCoinbase(trans fct.ITransaction) (bool, error) {
    b.BodyMR = nil
    if len(b.transactions)              != 0 ||
       len(trans.GetInputs())           != 0 || 
       len(trans.GetECOutputs())           != 0 ||
       len(trans.GetRCDs())             != 0 ||
       len(trans.GetSignatureBlocks())  != 0 {
        return false, fmt.Errorf("Cannot have inputs or EC outputs in the coinbase.")
    }

    // TODO Add check here for the proper payouts.
    
    b.transactions = append(b.transactions, trans)
    return true, nil
}
    

// Add a transaction to the Facoid block. If there is an error,
// then the transaction can be discarded.  If it returns true,
// then the transaction was added, if false it was not.
func (b *FBlock) AddTransaction(trans fct.ITransaction) (bool, error) {
	// These tests check that the Transaction itself is valid.  If it
	// is not internally valid, it never will be valid.
    b.BodyMR = nil
    valid := b.ValidateTransaction(trans)
	if !valid {
		return false, fmt.Errorf("Invalid Transaction")
	}
	
	// Check against address balances is done at the Factom level.

	b.transactions = append(b.transactions, trans)
	return true, nil
}

func (b FBlock) String() string {
    txt,err := b.MarshalText()
    if err != nil {return err.Error() }
    return string(txt)
}

// Marshal to text.  Largely a debugging thing.
func (b FBlock) MarshalText() (text []byte, err error) {
	var out bytes.Buffer

	out.WriteString("Transaction Block\n")
	out.WriteString("  ChainID:       ")
	out.WriteString(hex.EncodeToString(fct.FACTOID_CHAINID))
    if b.BodyMR == nil { b.BodyMR = new (fct.Hash) }
    out.WriteString("\n  BodyMR:        ")
    out.WriteString(b.BodyMR.String())
    if b.PrevKeyMR == nil { b.PrevKeyMR = new (fct.Hash) }
    out.WriteString("\n  PrevKeyMR:     ")
	out.WriteString(b.PrevKeyMR.String())
    if b.PrevFullHash == nil { b.PrevFullHash = new (fct.Hash) }
    out.WriteString("\n  PrevFullHash:  ")
	out.WriteString(b.PrevFullHash.String())
	out.WriteString("\n  ExchRate:      ")
	fct.WriteNumber64(&out, b.ExchRate)
	out.WriteString("\n  DBHeight:      ")
	fct.WriteNumber32(&out, b.DBHeight)
    out.WriteString("\n  #Transactions: ")
	fct.WriteNumber32(&out, uint32(len(b.transactions)))
	transdata, err := b.MarshalTrans()
	if err != nil {
		return out.Bytes(), err
	}
	out.WriteString("\n  Body Size:     ")
	fct.WriteNumber32(&out, uint32(len(transdata)))
	out.WriteString("\n\n")
    markPeriod := 0
   
	for i, trans := range b.transactions {
		
        for markPeriod < 10 && i == b.endOfPeriod[markPeriod] {
            out.WriteString(fmt.Sprintf("\n   End of Minute %d\n\n",markPeriod+1))
            markPeriod++
        }
        
        txt, err := trans.MarshalText()
		if err != nil {
			return out.Bytes(), err
		}
		out.Write(txt)
	}
	return out.Bytes(), nil
}

/**************************
 * Helper Functions
 **************************/

func NewFBlock(ExchRate uint64, DBHeight uint32) IFBlock {
	scb := new(FBlock)
    scb.BodyMR = new (fct.Hash)
    scb.PrevKeyMR  = new (fct.Hash)
    scb.PrevFullHash  = new (fct.Hash)
  	scb.ExchRate   = ExchRate
	scb.DBHeight   = DBHeight
	return scb
}
