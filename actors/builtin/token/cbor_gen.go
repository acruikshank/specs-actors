// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package token

import (
	"fmt"
	"io"

	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf

var lengthBufState = []byte{134}

func (t *State) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufState); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.Name (string) (string)
	if len(t.Name) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Name was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len(t.Name))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Name)); err != nil {
		return err
	}

	// t.Symbol (string) (string)
	if len(t.Symbol) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Symbol was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len(t.Symbol))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Symbol)); err != nil {
		return err
	}

	// t.Decimals (uint64) (uint64)

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.Decimals)); err != nil {
		return err
	}

	// t.TotalSupply (big.Int) (struct)
	if err := t.TotalSupply.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Balances (cid.Cid) (struct)

	if err := cbg.WriteCidBuf(scratch, w, t.Balances); err != nil {
		return xerrors.Errorf("failed to write cid field t.Balances: %w", err)
	}

	// t.Approvals (cid.Cid) (struct)

	if err := cbg.WriteCidBuf(scratch, w, t.Approvals); err != nil {
		return xerrors.Errorf("failed to write cid field t.Approvals: %w", err)
	}

	return nil
}

func (t *State) UnmarshalCBOR(r io.Reader) error {
	*t = State{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 6 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Name (string) (string)

	{
		sval, err := cbg.ReadStringBuf(br, scratch)
		if err != nil {
			return err
		}

		t.Name = string(sval)
	}
	// t.Symbol (string) (string)

	{
		sval, err := cbg.ReadStringBuf(br, scratch)
		if err != nil {
			return err
		}

		t.Symbol = string(sval)
	}
	// t.Decimals (uint64) (uint64)

	{

		maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Decimals = uint64(extra)

	}
	// t.TotalSupply (big.Int) (struct)

	{

		if err := t.TotalSupply.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.TotalSupply: %w", err)
		}

	}
	// t.Balances (cid.Cid) (struct)

	{

		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("failed to read cid field t.Balances: %w", err)
		}

		t.Balances = c

	}
	// t.Approvals (cid.Cid) (struct)

	{

		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("failed to read cid field t.Approvals: %w", err)
		}

		t.Approvals = c

	}
	return nil
}

var lengthBufConstructorParams = []byte{133}

func (t *ConstructorParams) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufConstructorParams); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.Name (string) (string)
	if len(t.Name) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Name was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len(t.Name))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Name)); err != nil {
		return err
	}

	// t.Symbol (string) (string)
	if len(t.Symbol) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Symbol was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len(t.Symbol))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Symbol)); err != nil {
		return err
	}

	// t.Decimals (uint64) (uint64)

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.Decimals)); err != nil {
		return err
	}

	// t.TotalSupply (big.Int) (struct)
	if err := t.TotalSupply.MarshalCBOR(w); err != nil {
		return err
	}

	// t.SystemAccount (address.Address) (struct)
	if err := t.SystemAccount.MarshalCBOR(w); err != nil {
		return err
	}
	return nil
}

func (t *ConstructorParams) UnmarshalCBOR(r io.Reader) error {
	*t = ConstructorParams{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 5 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Name (string) (string)

	{
		sval, err := cbg.ReadStringBuf(br, scratch)
		if err != nil {
			return err
		}

		t.Name = string(sval)
	}
	// t.Symbol (string) (string)

	{
		sval, err := cbg.ReadStringBuf(br, scratch)
		if err != nil {
			return err
		}

		t.Symbol = string(sval)
	}
	// t.Decimals (uint64) (uint64)

	{

		maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Decimals = uint64(extra)

	}
	// t.TotalSupply (big.Int) (struct)

	{

		if err := t.TotalSupply.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.TotalSupply: %w", err)
		}

	}
	// t.SystemAccount (address.Address) (struct)

	{

		if err := t.SystemAccount.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.SystemAccount: %w", err)
		}

	}
	return nil
}

var lengthBufTransferParams = []byte{130}

func (t *TransferParams) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufTransferParams); err != nil {
		return err
	}

	// t.To (address.Address) (struct)
	if err := t.To.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Value (big.Int) (struct)
	if err := t.Value.MarshalCBOR(w); err != nil {
		return err
	}
	return nil
}

func (t *TransferParams) UnmarshalCBOR(r io.Reader) error {
	*t = TransferParams{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 2 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.To (address.Address) (struct)

	{

		if err := t.To.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.To: %w", err)
		}

	}
	// t.Value (big.Int) (struct)

	{

		if err := t.Value.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.Value: %w", err)
		}

	}
	return nil
}

var lengthBufApproveParams = []byte{130}

func (t *ApproveParams) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufApproveParams); err != nil {
		return err
	}

	// t.Approvee (address.Address) (struct)
	if err := t.Approvee.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Value (big.Int) (struct)
	if err := t.Value.MarshalCBOR(w); err != nil {
		return err
	}
	return nil
}

func (t *ApproveParams) UnmarshalCBOR(r io.Reader) error {
	*t = ApproveParams{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 2 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Approvee (address.Address) (struct)

	{

		if err := t.Approvee.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.Approvee: %w", err)
		}

	}
	// t.Value (big.Int) (struct)

	{

		if err := t.Value.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.Value: %w", err)
		}

	}
	return nil
}
