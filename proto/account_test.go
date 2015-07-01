package proto

import (
	"testing"

	"euphoria.io/heim/proto/security"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewAccountSecurity(t *testing.T) {
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	unlock := func(sec *AccountSecurity, password string) (*security.ManagedKeyPair, error) {
		return sec.Unlock(security.KeyFromPasscode([]byte(password), sec.Nonce, sec.UserKey.KeyType))
	}

	Convey("Encryption and decryption of generated keys", t, func() {
		sec, clientKey, err := NewAccountSecurity(kms, "hunter2")
		So(err, ShouldBeNil)
		So(sec.SystemKey.Encrypted(), ShouldBeTrue)
		So(sec.UserKey.Encrypted(), ShouldBeTrue)
		So(sec.KeyPair.Encrypted(), ShouldBeTrue)
		So(len(sec.Nonce), ShouldEqual, sec.KeyPair.NonceSize())
		So(clientKey.Encrypted(), ShouldBeFalse)

		kek := sec.SystemKey.Clone()
		So(kms.DecryptKey(&kek), ShouldBeNil)

		skp := sec.KeyPair.Clone()
		So(skp.Decrypt(&kek), ShouldBeNil)

		kp, err := unlock(sec, "")
		So(err, ShouldEqual, ErrAccessDenied)
		So(kp, ShouldBeNil)

		kp, err = unlock(sec, "hunter2")
		So(err, ShouldBeNil)
		So(kp.PrivateKey, ShouldResemble, skp.PrivateKey)
	})

	Convey("Password resets", t, func() {
		sec, _, err := NewAccountSecurity(kms, "hunter2")
		So(err, ShouldBeNil)

		nsec, err := sec.ResetPassword(kms, "hunter3")
		So(err, ShouldBeNil)

		skp, err := unlock(sec, "hunter2")
		So(err, ShouldBeNil)

		_, err = unlock(nsec, "hunter2")
		So(err, ShouldEqual, ErrAccessDenied)

		kp, err := unlock(nsec, "hunter3")
		So(err, ShouldBeNil)
		So(kp.PrivateKey, ShouldResemble, skp.PrivateKey)
	})
}
