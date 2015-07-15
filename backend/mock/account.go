package mock

import (
	"fmt"

	"encoding/json"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

func NewAccount(kms security.KMS, password string) (proto.Account, *security.ManagedKey, error) {
	id, err := snowflake.New()
	if err != nil {
		return nil, nil, err
	}

	sec, clientKey, err := proto.NewAccountSecurity(kms, password)
	if err != nil {
		return nil, nil, err
	}

	account := &memAccount{
		id:  id,
		sec: *sec,
	}
	return account, clientKey, nil
}

type memAccount struct {
	id                 snowflake.Snowflake
	sec                proto.AccountSecurity
	staffCapability    security.Capability
	personalIdentities []proto.PersonalIdentity
}

func (a *memAccount) ID() snowflake.Snowflake { return a.id }

func (a *memAccount) KeyFromPassword(password string) *security.ManagedKey {
	return security.KeyFromPasscode([]byte(password), a.sec.Nonce, a.sec.UserKey.KeyType)
}

func (a *memAccount) KeyPair() security.ManagedKeyPair { return a.sec.KeyPair.Clone() }

func (a *memAccount) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	return a.sec.Unlock(clientKey)
}

func (a *memAccount) IsStaff() bool { return a.staffCapability != nil }

func (a *memAccount) UnlockStaffKMS(clientKey *security.ManagedKey) (security.KMS, error) {
	if a.staffCapability == nil {
		return nil, proto.ErrAccessDenied
	}

	key := a.sec.UserKey.Clone()
	if err := key.Decrypt(clientKey); err != nil {
		return nil, err
	}

	ssc := &security.SharedSecretCapability{Capability: a.staffCapability}
	data, err := ssc.DecryptPayload(&key)
	if err != nil {
		return nil, err
	}

	var kmsType security.KMSType
	if err := json.Unmarshal(ssc.PublicPayload(), &kmsType); err != nil {
		return nil, err
	}

	kmsCred, err := kmsType.KMSCredential()
	if err != nil {
		return nil, err
	}

	if err := kmsCred.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return kmsCred.KMS(), nil
}

func (a *memAccount) PersonalIdentities() []proto.PersonalIdentity { return a.personalIdentities }

type personalIdentity struct {
	accountID snowflake.Snowflake
	namespace string
	id        string
	verified  bool
}

func (pi *personalIdentity) Namespace() string { return pi.namespace }
func (pi *personalIdentity) ID() string        { return pi.id }
func (pi *personalIdentity) Verified() bool    { return pi.verified }

type accountManager struct {
	b *TestBackend
}

func (m *accountManager) Register(
	ctx scope.Context, kms security.KMS, namespace, id, password string,
	agentID string, agentKey *security.ManagedKey) (
	proto.Account, *security.ManagedKey, error) {

	m.b.Lock()
	defer m.b.Unlock()

	key := fmt.Sprintf("%s:%s", namespace, id)
	if _, ok := m.b.accountIDs[key]; ok {
		return nil, nil, proto.ErrPersonalIdentityInUse
	}

	account, clientKey, err := NewAccount(kms, password)
	if err != nil {
		return nil, nil, err
	}

	if m.b.accounts == nil {
		m.b.accounts = map[snowflake.Snowflake]proto.Account{account.ID(): account}
	} else {
		m.b.accounts[account.ID()] = account
	}

	pi := &personalIdentity{
		accountID: account.ID(),
		namespace: namespace,
		id:        id,
	}
	account.(*memAccount).personalIdentities = []proto.PersonalIdentity{pi}
	if m.b.accountIDs == nil {
		m.b.accountIDs = map[string]*personalIdentity{key: pi}
	} else {
		m.b.accountIDs[key] = pi
	}

	agent, err := m.b.AgentTracker().Get(ctx, agentID)
	if err != nil {
		backend.Logger(ctx).Printf(
			"error locating agent %s for new account %s:%s: %s", agentID, namespace, id, err)
	} else {
		if err := agent.SetClientKey(agentKey, clientKey); err != nil {
			backend.Logger(ctx).Printf(
				"error associating agent %s with new account %s:%s: %s", agentID, namespace, id, err)
		}
		agent.AccountID = account.ID().String()
	}

	return account, clientKey, nil
}

func (m *accountManager) Resolve(ctx scope.Context, namespace, id string) (proto.Account, error) {
	m.b.Lock()
	defer m.b.Unlock()

	key := fmt.Sprintf("%s:%s", namespace, id)
	pi, ok := m.b.accountIDs[key]
	if !ok {
		return nil, proto.ErrAccountNotFound
	}
	return m.b.accounts[pi.accountID], nil
}

func (m *accountManager) Get(ctx scope.Context, id snowflake.Snowflake) (proto.Account, error) {
	m.b.Lock()
	defer m.b.Unlock()

	account, ok := m.b.accounts[id]
	if !ok {
		return nil, proto.ErrAccountNotFound
	}
	return account, nil
}

func (m *accountManager) GrantStaff(
	ctx scope.Context, accountID snowflake.Snowflake, kmsCred security.KMSCredential) error {

	m.b.Lock()
	defer m.b.Unlock()

	account, ok := m.b.accounts[accountID]
	if !ok {
		return proto.ErrAccountNotFound
	}
	memAcc := account.(*memAccount)

	kms := kmsCred.KMS()
	key := memAcc.sec.SystemKey.Clone()
	if err := kms.DecryptKey(&key); err != nil {
		return err
	}

	nonce, err := kms.GenerateNonce(key.KeyType.BlockSize())
	if err != nil {
		return err
	}

	capability, err := security.GrantSharedSecretCapability(&key, nonce, kmsCred.KMSType(), kmsCred)
	if err != nil {
		return err
	}

	memAcc.staffCapability = capability
	return nil
}

func (m *accountManager) RevokeStaff(ctx scope.Context, accountID snowflake.Snowflake) error {
	m.b.Lock()
	defer m.b.Unlock()

	account, ok := m.b.accounts[accountID]
	if !ok {
		return proto.ErrAccountNotFound
	}
	memAcc := account.(*memAccount)
	memAcc.staffCapability = nil
	return nil
}
