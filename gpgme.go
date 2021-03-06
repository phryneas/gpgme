// Package gpgme provides a Go wrapper for the GPGME library
package gpgme

// #cgo LDFLAGS: -lgpgme -lassuan -lgpg-error
// #cgo CPPFLAGS: -D_FILE_OFFSET_BITS=64
// #include <stdlib.h>
// #include <gpgme.h>
// #include "go_gpgme.h"
import "C"
import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
	"unsafe"
)

var Version string

func init() {
	Version = C.GoString(C.gpgme_check_version(nil))
}

// Callback is the function that is called when a passphrase is required
type Callback func(uidHint string, prevWasBad bool, f *os.File) error

//export gogpgme_passfunc
func gogpgme_passfunc(hook unsafe.Pointer, uid_hint, passphrase_info *C.char, prev_was_bad, fd C.int) C.gpgme_error_t {
	c := callbackLookup(uintptr(hook)).(*Context)
	go_uid_hint := C.GoString(uid_hint)
	f := os.NewFile(uintptr(fd), go_uid_hint)
	defer f.Close()
	err := c.callback(go_uid_hint, prev_was_bad != 0, f)
	if err != nil {
		return C.GPG_ERR_CANCELED
	}
	return 0
}

type Protocol int

const (
	ProtocolOpenPGP  Protocol = C.GPGME_PROTOCOL_OpenPGP
	ProtocolCMS      Protocol = C.GPGME_PROTOCOL_CMS
	ProtocolGPGConf  Protocol = C.GPGME_PROTOCOL_GPGCONF
	ProtocolAssuan   Protocol = C.GPGME_PROTOCOL_ASSUAN
	ProtocolG13      Protocol = C.GPGME_PROTOCOL_G13
	ProtocolUIServer Protocol = C.GPGME_PROTOCOL_UISERVER
	ProtocolSpawn    Protocol = C.GPGME_PROTOCOL_SPAWN
	ProtocolDefault  Protocol = C.GPGME_PROTOCOL_DEFAULT
	ProtocolUnknown  Protocol = C.GPGME_PROTOCOL_UNKNOWN
)

type PinEntryMode int

const (
	PinEntryDefault  PinEntryMode = C.GPGME_PINENTRY_MODE_DEFAULT
	PinEntryAsk      PinEntryMode = C.GPGME_PINENTRY_MODE_ASK
	PinEntryCancel   PinEntryMode = C.GPGME_PINENTRY_MODE_CANCEL
	PinEntryError    PinEntryMode = C.GPGME_PINENTRY_MODE_ERROR
	PinEntryLoopback PinEntryMode = C.GPGME_PINENTRY_MODE_LOOPBACK
)

type EncryptFlag uint

const (
	EncryptAlwaysTrust EncryptFlag = C.GPGME_ENCRYPT_ALWAYS_TRUST
	EncryptNoEncryptTo EncryptFlag = C.GPGME_ENCRYPT_NO_ENCRYPT_TO
	EncryptPrepare     EncryptFlag = C.GPGME_ENCRYPT_PREPARE
	EncryptExceptSign  EncryptFlag = C.GPGME_ENCRYPT_EXPECT_SIGN
	EncryptNoCompress  EncryptFlag = C.GPGME_ENCRYPT_NO_COMPRESS
)

type HashAlgo int

// const values for HashAlgo values should be added when necessary.

type KeyListMode uint

const (
	KeyListModeLocal        KeyListMode = C.GPGME_KEYLIST_MODE_LOCAL
	KeyListModeExtern       KeyListMode = C.GPGME_KEYLIST_MODE_EXTERN
	KeyListModeSigs         KeyListMode = C.GPGME_KEYLIST_MODE_SIGS
	KeyListModeSigNotations KeyListMode = C.GPGME_KEYLIST_MODE_SIG_NOTATIONS
	KeyListModeWithSecret   KeyListMode = C.GPGME_KEYLIST_MODE_WITH_SECRET
	KeyListModeEphemeral    KeyListMode = C.GPGME_KEYLIST_MODE_EPHEMERAL
	KeyListModeModeValidate KeyListMode = C.GPGME_KEYLIST_MODE_VALIDATE
)

type PubkeyAlgo int

// const values for PubkeyAlgo values should be added when necessary.

type SigMode int

const (
	SigModeNormal SigMode = C.GPGME_SIG_MODE_NORMAL
	SigModeDetach SigMode = C.GPGME_SIG_MODE_DETACH
	SigModeClear  SigMode = C.GPGME_SIG_MODE_CLEAR
)

type SigSum int

const (
	SigSumValid      SigSum = C.GPGME_SIGSUM_VALID
	SigSumGreen      SigSum = C.GPGME_SIGSUM_GREEN
	SigSumRed        SigSum = C.GPGME_SIGSUM_RED
	SigSumKeyRevoked SigSum = C.GPGME_SIGSUM_KEY_REVOKED
	SigSumKeyExpired SigSum = C.GPGME_SIGSUM_KEY_EXPIRED
	SigSumSigExpired SigSum = C.GPGME_SIGSUM_SIG_EXPIRED
	SigSumKeyMissing SigSum = C.GPGME_SIGSUM_KEY_MISSING
	SigSumCRLMissing SigSum = C.GPGME_SIGSUM_CRL_MISSING
	SigSumCRLTooOld  SigSum = C.GPGME_SIGSUM_CRL_TOO_OLD
	SigSumBadPolicy  SigSum = C.GPGME_SIGSUM_BAD_POLICY
	SigSumSysError   SigSum = C.GPGME_SIGSUM_SYS_ERROR
)

type Validity int

const (
	ValidityUnknown   Validity = C.GPGME_VALIDITY_UNKNOWN
	ValidityUndefined Validity = C.GPGME_VALIDITY_UNDEFINED
	ValidityNever     Validity = C.GPGME_VALIDITY_NEVER
	ValidityMarginal  Validity = C.GPGME_VALIDITY_MARGINAL
	ValidityFull      Validity = C.GPGME_VALIDITY_FULL
	ValidityUltimate  Validity = C.GPGME_VALIDITY_ULTIMATE
)

type ErrorCode int

const (
	ErrorNoError ErrorCode = C.GPG_ERR_NO_ERROR
	ErrorEOF     ErrorCode = C.GPG_ERR_EOF
)

// Error is a wrapper for GPGME errors
type Error struct {
	err C.gpgme_error_t
}

func (e Error) Code() ErrorCode {
	return ErrorCode(C.gpgme_err_code(e.err))
}

func (e Error) Error() string {
	return C.GoString(C.gpgme_strerror(e.err))
}

func handleError(err C.gpgme_error_t) error {
	e := Error{err: err}
	if e.Code() == ErrorNoError {
		return nil
	}
	return e
}

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func EngineCheckVersion(p Protocol) error {
	return handleError(C.gpgme_engine_check_version(C.gpgme_protocol_t(p)))
}

type EngineInfo struct {
	info C.gpgme_engine_info_t
}

func (e *EngineInfo) Next() *EngineInfo {
	if e.info.next == nil {
		return nil
	}
	return &EngineInfo{info: e.info.next}
}

func (e *EngineInfo) Protocol() Protocol {
	return Protocol(e.info.protocol)
}

func (e *EngineInfo) FileName() string {
	return C.GoString(e.info.file_name)
}

func (e *EngineInfo) Version() string {
	return C.GoString(e.info.version)
}

func (e *EngineInfo) RequiredVersion() string {
	return C.GoString(e.info.req_version)
}

func (e *EngineInfo) HomeDir() string {
	return C.GoString(e.info.home_dir)
}

func GetEngineInfo() (*EngineInfo, error) {
	info := &EngineInfo{}
	return info, handleError(C.gpgme_get_engine_info(&info.info))
}

func SetEngineInfo(proto Protocol, fileName, homeDir string) error {
	var cfn, chome *C.char
	if fileName != "" {
		cfn = C.CString(fileName)
		defer C.free(unsafe.Pointer(cfn))
	}
	if homeDir != "" {
		chome = C.CString(homeDir)
		defer C.free(unsafe.Pointer(chome))
	}
	return handleError(C.gpgme_set_engine_info(C.gpgme_protocol_t(proto), cfn, chome))
}

func FindKeys(pattern string, secretOnly bool) ([]*Key, error) {
	var keys []*Key
	ctx, err := New()
	if err != nil {
		return keys, err
	}
	defer ctx.Release()
	if err := ctx.KeyListStart(pattern, secretOnly); err != nil {
		return keys, err
	}
	defer ctx.KeyListEnd()
	for ctx.KeyListNext() {
		keys = append(keys, ctx.Key)
	}
	if ctx.KeyError != nil {
		return keys, ctx.KeyError
	}
	return keys, nil
}

func Decrypt(r io.Reader) (*Data, error) {
	ctx, err := New()
	if err != nil {
		return nil, err
	}
	defer ctx.Release()
	cipher, err := NewDataReader(r)
	if err != nil {
		return nil, err
	}
	defer cipher.Close()
	plain, err := NewData()
	if err != nil {
		return nil, err
	}
	err = ctx.Decrypt(cipher, plain)
	plain.Seek(0, SeekSet)
	return plain, err
}

type Context struct {
	Key      *Key
	KeyError error

	callback Callback
	cbc      uintptr

	ctx C.gpgme_ctx_t
}

func New() (*Context, error) {
	c := &Context{}
	err := C.gpgme_new(&c.ctx)
	runtime.SetFinalizer(c, (*Context).Release)
	return c, handleError(err)
}

func (c *Context) Release() {
	if c.ctx == nil {
		return
	}
	if c.cbc > 0 {
		callbackDelete(c.cbc)
	}
	C.gpgme_release(c.ctx)
	c.ctx = nil
}

func (c *Context) SetArmor(yes bool) {
	C.gpgme_set_armor(c.ctx, cbool(yes))
}

func (c *Context) Armor() bool {
	return C.gpgme_get_armor(c.ctx) != 0
}

func (c *Context) SetTextMode(yes bool) {
	C.gpgme_set_textmode(c.ctx, cbool(yes))
}

func (c *Context) TextMode() bool {
	return C.gpgme_get_textmode(c.ctx) != 0
}

func (c *Context) SetProtocol(p Protocol) error {
	return handleError(C.gpgme_set_protocol(c.ctx, C.gpgme_protocol_t(p)))
}

func (c *Context) Protocol() Protocol {
	return Protocol(C.gpgme_get_protocol(c.ctx))
}

func (c *Context) SetKeyListMode(m KeyListMode) error {
	return handleError(C.gpgme_set_keylist_mode(c.ctx, C.gpgme_keylist_mode_t(m)))
}

func (c *Context) KeyListMode() KeyListMode {
	return KeyListMode(C.gpgme_get_keylist_mode(c.ctx))
}

func (c *Context) SetPinEntryMode(m PinEntryMode) error {
	return handleError(C.gpgme_set_pinentry_mode(c.ctx, C.gpgme_pinentry_mode_t(m)))
}

func (c *Context) PinEntryMode() PinEntryMode {
	return PinEntryMode(C.gpgme_get_pinentry_mode(c.ctx))
}

func (c *Context) SetCallback(callback Callback) error {
	var err error
	c.callback = callback
	if c.cbc > 0 {
		callbackDelete(c.cbc)
	}
	if callback != nil {
		cbc := callbackAdd(c)
		c.cbc = cbc
		_, err = C.gogpgme_set_passphrase_cb(c.ctx, C.gpgme_passphrase_cb_t(C.gogpgme_passfunc), C.uintptr_t(cbc))
	} else {
		c.cbc = 0
		_, err = C.gogpgme_set_passphrase_cb(c.ctx, nil, 0)
	}
	return err
}

func (c *Context) EngineInfo() *EngineInfo {
	return &EngineInfo{info: C.gpgme_ctx_get_engine_info(c.ctx)}
}

func (c *Context) SetEngineInfo(proto Protocol, fileName, homeDir string) error {
	var cfn, chome *C.char
	if fileName != "" {
		cfn = C.CString(fileName)
		defer C.free(unsafe.Pointer(cfn))
	}
	if homeDir != "" {
		chome = C.CString(homeDir)
		defer C.free(unsafe.Pointer(chome))
	}
	return handleError(C.gpgme_ctx_set_engine_info(c.ctx, C.gpgme_protocol_t(proto), cfn, chome))
}

func (c *Context) KeyListStart(pattern string, secretOnly bool) error {
	cpattern := C.CString(pattern)
	defer C.free(unsafe.Pointer(cpattern))
	err := C.gpgme_op_keylist_start(c.ctx, cpattern, cbool(secretOnly))
	return handleError(err)
}

func (c *Context) KeyListNext() bool {
	c.Key = newKey()
	err := handleError(C.gpgme_op_keylist_next(c.ctx, &c.Key.k))
	if err != nil {
		if e, ok := err.(Error); ok && e.Code() == ErrorEOF {
			c.KeyError = nil
		} else {
			c.KeyError = err
		}
		return false
	}
	c.KeyError = nil
	return true
}

func (c *Context) KeyListEnd() error {
	return handleError(C.gpgme_op_keylist_end(c.ctx))
}

func (c *Context) GetKey(fingerprint string, secret bool) (*Key, error) {
	key := newKey()
	cfpr := C.CString(fingerprint)
	defer C.free(unsafe.Pointer(cfpr))
	err := handleError(C.gpgme_get_key(c.ctx, cfpr, &key.k, cbool(secret)))
	if e, ok := err.(Error); key.k == nil && ok && e.Code() == ErrorEOF {
		return nil, fmt.Errorf("key %q not found", fingerprint)
	}
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (c *Context) Decrypt(ciphertext, plaintext *Data) error {
	return handleError(C.gpgme_op_decrypt(c.ctx, ciphertext.dh, plaintext.dh))
}

func (c *Context) DecryptVerify(ciphertext, plaintext *Data) error {
	return handleError(C.gpgme_op_decrypt_verify(c.ctx, ciphertext.dh, plaintext.dh))
}

type Signature struct {
	Summary        SigSum
	Fingerprint    string
	Status         error
	Timestamp      time.Time
	ExpTimestamp   time.Time
	WrongKeyUsage  bool
	PKATrust       uint
	ChainModel     bool
	Validity       Validity
	ValidityReason error
	PubkeyAlgo     PubkeyAlgo
	HashAlgo       HashAlgo
}

func (c *Context) Verify(sig, signedText, plain *Data) (string, []Signature, error) {
	var signedTextPtr, plainPtr C.gpgme_data_t = nil, nil
	if signedText != nil {
		signedTextPtr = signedText.dh
	}
	if plain != nil {
		plainPtr = plain.dh
	}
	err := handleError(C.gpgme_op_verify(c.ctx, sig.dh, signedTextPtr, plainPtr))
	if err != nil {
		return "", nil, err
	}
	res := C.gpgme_op_verify_result(c.ctx)
	sigs := []Signature{}
	for s := res.signatures; s != nil; s = s.next {
		sig := Signature{
			Summary:     SigSum(s.summary),
			Fingerprint: C.GoString(s.fpr),
			Status:      handleError(s.status),
			// s.notations not implemented
			Timestamp:      time.Unix(int64(s.timestamp), 0),
			ExpTimestamp:   time.Unix(int64(s.exp_timestamp), 0),
			WrongKeyUsage:  C.signature_wrong_key_usage(s) != 0,
			PKATrust:       uint(C.signature_pka_trust(s)),
			ChainModel:     C.signature_chain_model(s) != 0,
			Validity:       Validity(s.validity),
			ValidityReason: handleError(s.validity_reason),
			PubkeyAlgo:     PubkeyAlgo(s.pubkey_algo),
			HashAlgo:       HashAlgo(s.hash_algo),
		}
		sigs = append(sigs, sig)
	}
	return C.GoString(res.file_name), sigs, nil
}

func (c *Context) Encrypt(recipients []*Key, flags EncryptFlag, plaintext, ciphertext *Data) error {
	size := unsafe.Sizeof(new(C.gpgme_key_t))
	recp := C.calloc(C.size_t(len(recipients)+1), C.size_t(size))
	defer C.free(recp)
	for i := range recipients {
		ptr := (*C.gpgme_key_t)(unsafe.Pointer(uintptr(recp) + size*uintptr(i)))
		*ptr = recipients[i].k
	}
	err := C.gpgme_op_encrypt(c.ctx, (*C.gpgme_key_t)(recp), C.gpgme_encrypt_flags_t(flags), plaintext.dh, ciphertext.dh)
	return handleError(err)
}

func (c *Context) Sign(signers []*Key, plain, sig *Data, mode SigMode) error {
	C.gpgme_signers_clear(c.ctx)
	for _, k := range signers {
		if err := handleError(C.gpgme_signers_add(c.ctx, k.k)); err != nil {
			C.gpgme_signers_clear(c.ctx)
			return err
		}
	}
	return handleError(C.gpgme_op_sign(c.ctx, plain.dh, sig.dh, C.gpgme_sig_mode_t(mode)))
}

type AssuanDataCallback func(data []byte) error
type AssuanInquireCallback func(name, args string) error
type AssuanStatusCallback func(status, args string) error

// AssuanSend sends a raw Assuan command to gpg-agent
func (c *Context) AssuanSend(
	cmd string,
	data AssuanDataCallback,
	inquiry AssuanInquireCallback,
	status AssuanStatusCallback,
) error {
	var operr C.gpgme_error_t

	dataPtr := callbackAdd(&data)
	inquiryPtr := callbackAdd(&inquiry)
	statusPtr := callbackAdd(&status)
	cmdCStr := C.CString(cmd)
	defer C.free(unsafe.Pointer(cmdCStr))
	err := C.gogpgme_op_assuan_transact_ext(
		c.ctx,
		cmdCStr,
		C.uintptr_t(dataPtr),
		C.uintptr_t(inquiryPtr),
		C.uintptr_t(statusPtr),
		&operr,
	)

	if handleError(operr) != nil {
		return handleError(operr)
	}
	return handleError(err)
}

//export gogpgme_assuan_data_callback
func gogpgme_assuan_data_callback(handle unsafe.Pointer, data unsafe.Pointer, datalen C.size_t) C.gpgme_error_t {
	c := callbackLookup(uintptr(handle)).(*AssuanDataCallback)
	if *c == nil {
		return 0
	}
	(*c)(C.GoBytes(data, C.int(datalen)))
	return 0
}

//export gogpgme_assuan_inquiry_callback
func gogpgme_assuan_inquiry_callback(handle unsafe.Pointer, cName *C.char, cArgs *C.char) C.gpgme_error_t {
	name := C.GoString(cName)
	args := C.GoString(cArgs)
	c := callbackLookup(uintptr(handle)).(*AssuanInquireCallback)
	if *c == nil {
		return 0
	}
	(*c)(name, args)
	return 0
}

//export gogpgme_assuan_status_callback
func gogpgme_assuan_status_callback(handle unsafe.Pointer, cStatus *C.char, cArgs *C.char) C.gpgme_error_t {
	status := C.GoString(cStatus)
	args := C.GoString(cArgs)
	c := callbackLookup(uintptr(handle)).(*AssuanStatusCallback)
	if *c == nil {
		return 0
	}
	(*c)(status, args)
	return 0
}

// ExportModeFlags defines how keys are exported from Export
type ExportModeFlags uint

const (
	ExportModeExtern  ExportModeFlags = C.GPGME_EXPORT_MODE_EXTERN
	ExportModeMinimal ExportModeFlags = C.GPGME_EXPORT_MODE_MINIMAL
	ExportModeSecret  ExportModeFlags = C.GPGME_EXPORT_MODE_SECRET
	ExportModeRaw     ExportModeFlags = C.GPGME_EXPORT_MODE_RAW
	ExportModePKCS12  ExportModeFlags = C.GPGME_EXPORT_MODE_PKCS12
)

func (c *Context) Export(pattern string, mode ExportModeFlags, data *Data) error {
	pat := C.CString(pattern)
	defer C.free(unsafe.Pointer(pat))
	return handleError(C.gpgme_op_export(c.ctx, pat, C.gpgme_export_mode_t(mode), data.dh))
}

// ImportStatusFlags describes the type of ImportStatus.Status. The C API in gpgme.h simply uses "unsigned".
type ImportStatusFlags uint

const (
	ImportNew    ImportStatusFlags = C.GPGME_IMPORT_NEW
	ImportUID    ImportStatusFlags = C.GPGME_IMPORT_UID
	ImportSIG    ImportStatusFlags = C.GPGME_IMPORT_SIG
	ImportSubKey ImportStatusFlags = C.GPGME_IMPORT_SUBKEY
	ImportSecret ImportStatusFlags = C.GPGME_IMPORT_SECRET
)

type ImportStatus struct {
	Fingerprint string
	Result      error
	Status      ImportStatusFlags
}

type ImportResult struct {
	Considered      int
	NoUserID        int
	Imported        int
	ImportedRSA     int
	Unchanged       int
	NewUserIDs      int
	NewSubKeys      int
	NewSignatures   int
	NewRevocations  int
	SecretRead      int
	SecretImported  int
	SecretUnchanged int
	NotImported     int
	Imports         []ImportStatus
}

func (c *Context) Import(keyData *Data) (*ImportResult, error) {
	err := handleError(C.gpgme_op_import(c.ctx, keyData.dh))
	if err != nil {
		return nil, err
	}
	res := C.gpgme_op_import_result(c.ctx)
	imports := []ImportStatus{}
	for s := res.imports; s != nil; s = s.next {
		imports = append(imports, ImportStatus{
			Fingerprint: C.GoString(s.fpr),
			Result:      handleError(s.result),
			Status:      ImportStatusFlags(s.status),
		})
	}
	return &ImportResult{
		Considered:      int(res.considered),
		NoUserID:        int(res.no_user_id),
		Imported:        int(res.imported),
		ImportedRSA:     int(res.imported_rsa),
		Unchanged:       int(res.unchanged),
		NewUserIDs:      int(res.new_user_ids),
		NewSubKeys:      int(res.new_sub_keys),
		NewSignatures:   int(res.new_signatures),
		NewRevocations:  int(res.new_revocations),
		SecretRead:      int(res.secret_read),
		SecretImported:  int(res.secret_imported),
		SecretUnchanged: int(res.secret_unchanged),
		NotImported:     int(res.not_imported),
		Imports:         imports,
	}, nil
}

type Key struct {
	k C.gpgme_key_t
}

func newKey() *Key {
	k := &Key{}
	runtime.SetFinalizer(k, (*Key).Release)
	return k
}

func (k *Key) Release() {
	C.gpgme_key_release(k.k)
	k.k = nil
}

func (k *Key) Revoked() bool {
	return C.key_revoked(k.k) != 0
}

func (k *Key) Expired() bool {
	return C.key_expired(k.k) != 0
}

func (k *Key) Disabled() bool {
	return C.key_disabled(k.k) != 0
}

func (k *Key) Invalid() bool {
	return C.key_invalid(k.k) != 0
}

func (k *Key) CanEncrypt() bool {
	return C.key_can_encrypt(k.k) != 0
}

func (k *Key) CanSign() bool {
	return C.key_can_sign(k.k) != 0
}

func (k *Key) CanCertify() bool {
	return C.key_can_certify(k.k) != 0
}

func (k *Key) Secret() bool {
	return C.key_secret(k.k) != 0
}

func (k *Key) CanAuthenticate() bool {
	return C.key_can_authenticate(k.k) != 0
}

func (k *Key) IsQualified() bool {
	return C.key_is_qualified(k.k) != 0
}

func (k *Key) Protocol() Protocol {
	return Protocol(k.k.protocol)
}

func (k *Key) IssuerSerial() string {
	return C.GoString(k.k.issuer_serial)
}

func (k *Key) IssuerName() string {
	return C.GoString(k.k.issuer_name)
}

func (k *Key) ChainID() string {
	return C.GoString(k.k.chain_id)
}

func (k *Key) OwnerTrust() Validity {
	return Validity(k.k.owner_trust)
}

func (k *Key) SubKeys() *SubKey {
	if k.k.subkeys == nil {
		return nil
	}
	return &SubKey{k: k.k.subkeys, parent: k}
}

func (k *Key) UserIDs() *UserID {
	if k.k.uids == nil {
		return nil
	}
	return &UserID{u: k.k.uids, parent: k}
}

func (k *Key) KeyListMode() KeyListMode {
	return KeyListMode(k.k.keylist_mode)
}

type SubKey struct {
	k      C.gpgme_subkey_t
	parent *Key // make sure the key is not released when we have a reference to a subkey
}

func (k *SubKey) Next() *SubKey {
	if k.k.next == nil {
		return nil
	}
	return &SubKey{k: k.k.next, parent: k.parent}
}

func (k *SubKey) Revoked() bool {
	return C.subkey_revoked(k.k) != 0
}

func (k *SubKey) Expired() bool {
	return C.subkey_expired(k.k) != 0
}

func (k *SubKey) Disabled() bool {
	return C.subkey_disabled(k.k) != 0
}

func (k *SubKey) Invalid() bool {
	return C.subkey_invalid(k.k) != 0
}

func (k *SubKey) Secret() bool {
	return C.subkey_secret(k.k) != 0
}

func (k *SubKey) KeyID() string {
	return C.GoString(k.k.keyid)
}

func (k *SubKey) Fingerprint() string {
	return C.GoString(k.k.fpr)
}

func (k *SubKey) Created() time.Time {
	if k.k.timestamp <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(k.k.timestamp), 0)
}

func (k *SubKey) Expires() time.Time {
	if k.k.expires <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(k.k.expires), 0)
}

func (k *SubKey) CardNumber() string {
	return C.GoString(k.k.card_number)
}

type UserID struct {
	u      C.gpgme_user_id_t
	parent *Key // make sure the key is not released when we have a reference to a user ID
}

func (u *UserID) Next() *UserID {
	if u.u.next == nil {
		return nil
	}
	return &UserID{u: u.u.next, parent: u.parent}
}

func (u *UserID) Revoked() bool {
	return C.uid_revoked(u.u) != 0
}

func (u *UserID) Invalid() bool {
	return C.uid_invalid(u.u) != 0
}

func (u *UserID) Validity() Validity {
	return Validity(u.u.validity)
}

func (u *UserID) UID() string {
	return C.GoString(u.u.uid)
}

func (u *UserID) Name() string {
	return C.GoString(u.u.name)
}

func (u *UserID) Comment() string {
	return C.GoString(u.u.comment)
}

func (u *UserID) Email() string {
	return C.GoString(u.u.email)
}

// This is somewhat of a horrible hack. We need to unset GPG_AGENT_INFO so that gpgme does not pass --use-agent to GPG.
// os.Unsetenv should be enough, but that only calls the underlying C library (which gpgme uses) if cgo is involved
// - and cgo can't be used in tests. So, provide this helper for test initialization.
func unsetenvGPGAgentInfo() {
	v := C.CString("GPG_AGENT_INFO")
	defer C.free(unsafe.Pointer(v))
	C.unsetenv(v)
}
