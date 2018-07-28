package channeldb

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/wakiyamap/lnd/keychain"
	"github.com/wakiyamap/lnd/lnwire"
	"github.com/wakiyamap/lnd/shachain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// outPointSize is the size of a serialized outpoint on disk.
const outPointSize = 36

// writeOutpoint writes an outpoint to the passed writer using the minimal
// amount of bytes possible.
func writeOutpoint(w io.Writer, o *wire.OutPoint) error {
	if _, err := w.Write(o.Hash[:]); err != nil {
		return err
	}
	if err := binary.Write(w, byteOrder, o.Index); err != nil {
		return err
	}

	return nil
}

// readOutpoint reads an outpoint from the passed reader that was previously
// written using the writeOutpoint struct.
func readOutpoint(r io.Reader, o *wire.OutPoint) error {
	if _, err := io.ReadFull(r, o.Hash[:]); err != nil {
		return err
	}
	if err := binary.Read(r, byteOrder, &o.Index); err != nil {
		return err
	}

	return nil
}

// UnknownElementType is an error returned when the codec is unable to encode or
// decode a particular type.
type UnknownElementType struct {
	method  string
	element interface{}
}

// Error returns the name of the method that encountered the error, as well as
// the type that was unsupported.
func (e UnknownElementType) Error() string {
	return fmt.Sprintf("Unknown type in %s: %T", e.method, e.element)
}

// WriteElement is a one-stop shop to write the big endian representation of
// any element which is to be serialized for storage on disk. The passed
// io.Writer should be backed by an appropriately sized byte slice, or be able
// to dynamically expand to accommodate additional data.
func WriteElement(w io.Writer, element interface{}) error {
	switch e := element.(type) {
	case keychain.KeyDescriptor:
		if err := binary.Write(w, byteOrder, e.Family); err != nil {
			return err
		}
		if err := binary.Write(w, byteOrder, e.Index); err != nil {
			return err
		}

		if e.PubKey != nil {
			if err := binary.Write(w, byteOrder, true); err != nil {
			}

			return WriteElement(w, e.PubKey)
		}

		return binary.Write(w, byteOrder, false)
	case ChannelType:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case chainhash.Hash:
		if _, err := w.Write(e[:]); err != nil {
			return err
		}

	case wire.OutPoint:
		return writeOutpoint(w, &e)

	case lnwire.ShortChannelID:
		if err := binary.Write(w, byteOrder, e.ToUint64()); err != nil {
			return err
		}

	case uint64:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case uint32:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case int32:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case uint16:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case bool:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case btcutil.Amount:
		if err := binary.Write(w, byteOrder, uint64(e)); err != nil {
			return err
		}

	case lnwire.MilliSatoshi:
		if err := binary.Write(w, byteOrder, uint64(e)); err != nil {
			return err
		}

	case *btcec.PublicKey:
		b := e.SerializeCompressed()
		if _, err := w.Write(b); err != nil {
			return err
		}

	case shachain.Producer:
		return e.Encode(w)

	case shachain.Store:
		return e.Encode(w)

	case *wire.MsgTx:
		return e.Serialize(w)

	case [32]byte:
		if _, err := w.Write(e[:]); err != nil {
			return err
		}

	case []byte:
		if err := wire.WriteVarBytes(w, 0, e); err != nil {
			return err
		}

	case lnwire.Message:
		if _, err := lnwire.WriteMessage(w, e, 0); err != nil {
			return err
		}

	case ChannelStatus:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case ClosureType:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case lnwire.FundingFlag:
		if err := binary.Write(w, byteOrder, e); err != nil {
			return err
		}

	case net.Addr:
		if err := serializeAddr(w, e); err != nil {
			return err
		}

	case []net.Addr:
		if err := WriteElement(w, uint32(len(e))); err != nil {
			return err
		}

		for _, addr := range e {
			if err := serializeAddr(w, addr); err != nil {
				return err
			}
		}

	default:
		return UnknownElementType{"WriteElement", e}
	}

	return nil
}

// WriteElements is writes each element in the elements slice to the passed
// io.Writer using WriteElement.
func WriteElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := WriteElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadElement is a one-stop utility function to deserialize any datastructure
// encoded using the serialization format of the database.
func ReadElement(r io.Reader, element interface{}) error {
	switch e := element.(type) {
	case *keychain.KeyDescriptor:
		if err := binary.Read(r, byteOrder, &e.Family); err != nil {
			return err
		}
		if err := binary.Read(r, byteOrder, &e.Index); err != nil {
			return err
		}

		var hasPubKey bool
		if err := binary.Read(r, byteOrder, &hasPubKey); err != nil {
			return err
		}

		if hasPubKey {
			return ReadElement(r, &e.PubKey)
		}

	case *ChannelType:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *chainhash.Hash:
		if _, err := io.ReadFull(r, e[:]); err != nil {
			return err
		}

	case *wire.OutPoint:
		return readOutpoint(r, e)

	case *lnwire.ShortChannelID:
		var a uint64
		if err := binary.Read(r, byteOrder, &a); err != nil {
			return err
		}
		*e = lnwire.NewShortChanIDFromInt(a)

	case *uint64:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *uint32:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *int32:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *uint16:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *bool:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *btcutil.Amount:
		var a uint64
		if err := binary.Read(r, byteOrder, &a); err != nil {
			return err
		}

		*e = btcutil.Amount(a)

	case *lnwire.MilliSatoshi:
		var a uint64
		if err := binary.Read(r, byteOrder, &a); err != nil {
			return err
		}

		*e = lnwire.MilliSatoshi(a)

	case **btcec.PublicKey:
		var b [btcec.PubKeyBytesLenCompressed]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return err
		}

		pubKey, err := btcec.ParsePubKey(b[:], btcec.S256())
		if err != nil {
			return err
		}
		*e = pubKey

	case *shachain.Producer:
		var root [32]byte
		if _, err := io.ReadFull(r, root[:]); err != nil {
			return err
		}

		// TODO(roasbeef): remove
		producer, err := shachain.NewRevocationProducerFromBytes(root[:])
		if err != nil {
			return err
		}

		*e = producer

	case *shachain.Store:
		store, err := shachain.NewRevocationStoreFromBytes(r)
		if err != nil {
			return err
		}

		*e = store

	case **wire.MsgTx:
		tx := wire.NewMsgTx(2)
		if err := tx.Deserialize(r); err != nil {
			return err
		}

		*e = tx

	case *[32]byte:
		if _, err := io.ReadFull(r, e[:]); err != nil {
			return err
		}

	case *[]byte:
		bytes, err := wire.ReadVarBytes(r, 0, 66000, "[]byte")
		if err != nil {
			return err
		}

		*e = bytes

	case *lnwire.Message:
		msg, err := lnwire.ReadMessage(r, 0)
		if err != nil {
			return err
		}

		*e = msg

	case *ChannelStatus:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *ClosureType:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *lnwire.FundingFlag:
		if err := binary.Read(r, byteOrder, e); err != nil {
			return err
		}

	case *net.Addr:
		addr, err := deserializeAddr(r)
		if err != nil {
			return err
		}
		*e = addr

	case *[]net.Addr:
		var numAddrs uint32
		if err := ReadElement(r, &numAddrs); err != nil {
			return err
		}

		*e = make([]net.Addr, numAddrs)
		for i := uint32(0); i < numAddrs; i++ {
			addr, err := deserializeAddr(r)
			if err != nil {
				return err
			}
			(*e)[i] = addr
		}

	default:
		return UnknownElementType{"ReadElement", e}
	}

	return nil
}

// ReadElements deserializes a variable number of elements into the passed
// io.Reader, with each element being deserialized according to the ReadElement
// function.
func ReadElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := ReadElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}
