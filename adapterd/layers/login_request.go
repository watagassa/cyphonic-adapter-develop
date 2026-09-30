// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"os"
)

// LoginRequest is Login Request of a CYPHONIC packet.
// AuthType sets the authentication method for authentication service.
// ExecMode handles whether you are a user or an administrator.
type LoginRequest struct {
	BaseHeader        BaseHeader
	ApplicationID     ApplicationID
	AuthType          TypeAuthClass
	ExecMode          TypeExecModeClass
	DeviceIDLength    uint16
	PasswordLength    uint16
	CertificateLength uint16
	DeviceID          DeviceID
	Password          Password
	Certificate       Certificate
}

// ApplicationID is a CYPHONIC Application ID, a slice of bytes.
// length of the byte slice: a 16-byte slice.
type ApplicationID []byte

// Payload length in LoginRequest.
const (
	// ApplicationIDSize lengths (bytes).
	ApplicationIDSize = 16

	// AuthTypeSize lengths (bytes).
	AuthTypeSize = 1

	// ExecModeSize lengths (bytes).
	ExecModeSize = 1

	// DeviceIDLengthSize lengths (bytes).
	DeviceIDLengthSize = 2

	// PasswordLengthSize lengths (bytes).
	PasswordLengthSize = 2

	// CertificateLengthSize lengths (bytes).
	CertificateLengthSize = 2
)

// BaseLoginRequestLength is length of (base) body part of LoginRequest.
// Also, since the length of DeviceID and Password or Certificate is variable length, when
// marshaling, calculate and add.
const BaseLoginRequestLength = BaseHeaderLen + ApplicationIDSize + AuthTypeSize + ExecModeSize + DeviceIDLengthSize + PasswordLengthSize + CertificateLengthSize

// DeviceID is the device ID for logging in.
type DeviceID []byte

// Password is the password to login.
type Password []byte

// SHA512Password is the sha256 password to login.
type SHA512Password [64]byte

// Certificate is the certificate to login.
type Certificate []byte

// TypeAuthClass defines the class associated with Auth type in CYPHONIC packets.
type TypeAuthClass uint8

// TypeExecModeClass defines the class associated with the execution mode of CYPHONIC packets.
type TypeExecModeClass uint8

// TypeAuthClass known values.
const (
	TypeDeviceIDPassword      TypeAuthClass = 1
	TypeDigitalAuthentication TypeAuthClass = 2
	TypeSingleSignOn          TypeAuthClass = 3
)

// AuthenticationElement stores information required for authentication.
// Also, unused fields are filled with empty.
type AuthenticationElement struct {
	AuthType       TypeAuthClass
	DeviceID       DeviceID
	HashedPassword SHA512Password
	Certificate    Certificate
}

// SerializeLoginRequestPacketLength sets the length of the Login Request packet.
func SerializeLoginRequestPacketLength(lreq *LoginRequest, valiableLength uint16) uint16 {
	lreq.BaseHeader.MessageLength = BaseLoginRequestLength + valiableLength

	return lreq.BaseHeader.MessageLength
}

// SerializeAuthType serializes the auth type.
func SerializeAuthType(login *LoginRequest, authtype TypeAuthClass) TypeAuthClass {
	login.AuthType = authtype

	return login.AuthType
}

// SerializeExecMode serializes the crypt mode.
func SerializeExecMode(login *LoginRequest) TypeExecModeClass {
	login.ExecMode = TypeExecModeClass(0)

	return login.ExecMode
}

// SerializeDeviceIDLength serializes the deviceID length.
func SerializeDeviceIDLength(login *LoginRequest, deviceID []byte) uint16 {
	login.DeviceIDLength = uint16(len(deviceID))

	return login.DeviceIDLength
}

// SerializePasswordLength serializes the crypt mode.
func SerializePasswordLength(login *LoginRequest, password [64]byte) uint16 {
	login.PasswordLength = uint16(len(password))

	return login.PasswordLength
}

// SerializeCertifiacateLength serializes the certificate length.
func SerializeCertifiacateLength(login *LoginRequest, certificate []byte) uint16 {
	login.CertificateLength = uint16(len(certificate))

	return login.CertificateLength
}

// SetDeviceID serializes the deviceID.
func SetDeviceID(login *LoginRequest, deviceID []byte) uint16 {
	login.DeviceID = deviceID

	return uint16(len(login.DeviceID))
}

// SetPassword serializes the password.
func SetPassword(login *LoginRequest, password [64]byte) uint16 {
	login.Password = password[:]

	return uint16(len(login.Password))
}

// SetCertificate serializes the certificate.
func SetCertificate(login *LoginRequest, certificate []byte) uint16 {
	login.Certificate = certificate

	return uint16(len(login.Certificate))
}

// GenerateHashedPassword returns the hashed password of sha512.
func GenerateHashedPassword(password []byte) [64]byte {
	sha512 := sha512.Sum512(password)

	return sha512
}

// SetAuthenticationElement sets the authentication elements.
func SetAuthenticationElement(authType int, deviceID []byte, hashedPassword [64]byte, certificatePath string) (*AuthenticationElement, error) {
	authElem := new(AuthenticationElement)
	authElem.AuthType = TypeAuthClass(authType)

	switch authElem.AuthType {
	case TypeDeviceIDPassword:
		authElem.DeviceID = DeviceID(deviceID)
		authElem.HashedPassword = SHA512Password(hashedPassword)
	case TypeDigitalAuthentication:
		certificate, err := os.ReadFile(certificatePath)
		if err != nil {
			return nil, err
		}

		authElem.Certificate = Certificate(certificate)
	case TypeSingleSignOn:
	}

	return authElem, nil
}

// HandleLoginPattern handles login patterns.
func (login *LoginRequest) HandleLoginPattern(authElem *AuthenticationElement) error {
	switch authElem.AuthType {
	case TypeDeviceIDPassword:
		if err := login.GenerateDeviceIDPasswordLoginRequest(authElem.DeviceID, authElem.HashedPassword); err != nil {
			return err
		}
	case TypeDigitalAuthentication:
		if err := login.GenerateDigitalAuthenticationLoginRequest(authElem.Certificate); err != nil {
			return err
		}
	case TypeSingleSignOn:
	}

	return nil
}

// GenerateApplicationID is to generate CYPHONIC APP ID.
// Application ID is 128bit length (16 Byte).
func GenerateApplicationID(login *LoginRequest) {
	// FIXME: ApplicationID is fixed.
	login.ApplicationID = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
}

// GenerateDeviceIDPasswordLoginRequest generates Login Request packets using DeviceID and Password.
func (login *LoginRequest) GenerateDeviceIDPasswordLoginRequest(deviceID []byte, password [64]byte) error {
	GenerateApplicationID(login)
	SerializeAuthType(login, TypeDeviceIDPassword)
	SerializeExecMode(login)
	SerializeDeviceIDLength(login, deviceID)
	SerializePasswordLength(login, password)
	SetDeviceID(login, deviceID)
	SetPassword(login, password)

	return nil
}

// GenerateDigitalAuthenticationLoginRequest generates Login Request packets using Certificate.
func (login *LoginRequest) GenerateDigitalAuthenticationLoginRequest(certificate Certificate) error {
	GenerateApplicationID(login)
	SerializeAuthType(login, TypeDigitalAuthentication)
	SerializeExecMode(login)
	SerializeCertifiacateLength(login, certificate)
	SetCertificate(login, certificate)

	return nil
}

// Marshal returns binary data.
func (login *LoginRequest) Marshal() ([]byte, error) {
	length := login.BaseHeader.MessageLength - (login.DeviceIDLength + login.PasswordLength + login.CertificateLength)

	if (len(login.DeviceID) != int(login.DeviceIDLength)) && (len(login.Password) != int(login.PasswordLength)) {
		err := errors.New("deviceID, password does not match DeviceIDLength, PasswordLength")
		return nil, err
	} else if len(login.Certificate) != int(login.CertificateLength) {
		err := errors.New("certificate does not match CertificateLength")
		return nil, err
	}

	buf := make([]byte, length)

	// BaseHeader
	binary.BigEndian.PutUint32(buf[0:4], login.BaseHeader.TransactionID)
	buf[4] = login.BaseHeader.Version
	buf[5] = login.BaseHeader.Flag
	buf[6] = login.BaseHeader.Type
	buf[7] = login.BaseHeader.Count
	binary.BigEndian.PutUint32(buf[8:12], login.BaseHeader.SequenceNumber)
	binary.BigEndian.PutUint16(buf[12:14], login.BaseHeader.MessageLength)
	buf[14] = login.BaseHeader.NextOpt
	buf[15] = login.BaseHeader.Reserved
	binary.BigEndian.PutUint64(buf[16:24], binary.BigEndian.Uint64(login.BaseHeader.ID[0:8]))
	binary.BigEndian.PutUint64(buf[24:32], binary.BigEndian.Uint64(login.BaseHeader.ID[8:16]))
	// Login Request Body
	binary.BigEndian.PutUint64(buf[32:40], binary.BigEndian.Uint64(login.ApplicationID[0:8]))
	binary.BigEndian.PutUint64(buf[40:48], binary.BigEndian.Uint64(login.ApplicationID[8:16]))
	buf[48] = uint8(login.AuthType)
	buf[49] = uint8(login.ExecMode)
	binary.BigEndian.PutUint16(buf[50:52], login.DeviceIDLength)
	binary.BigEndian.PutUint16(buf[52:54], login.PasswordLength)
	binary.BigEndian.PutUint16(buf[54:56], login.CertificateLength)

	switch login.AuthType {
	case TypeDeviceIDPassword:
		buf = append(buf, login.DeviceID...)
		buf = append(buf, login.Password...)
	case TypeDigitalAuthentication:
		buf = append(buf, login.Certificate...)
	case TypeSingleSignOn:
	}

	if len(buf) != int(login.BaseHeader.MessageLength) {
		err := errors.New("message does not match MessageLength")
		return nil, err
	}

	return buf, nil
}
