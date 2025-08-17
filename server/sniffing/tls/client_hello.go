package tls

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/cryptobyte"
)

// scsvRenegotiation 定义重协商的信号密码套件值（SCSV）
const (
	scsvRenegotiation uint16 = 0x00ff
)

// keyShare 结构体表示密钥共享信息
type keyShare struct {
	group CurveID // 曲线 ID
	data  []byte  // 密钥共享数据
}

// pskIdentity 结构体表示预共享密钥（PSK）的身份信息
type pskIdentity struct {
	label               []byte // 标签
	obfuscatedTicketAge uint32 // 混淆后的票据年龄
}

// CurveID 类型表示曲线的 ID
type CurveID uint16

// 定义各种支持的曲线 ID
const (
	CurveP256 CurveID = 23
	CurveP384 CurveID = 24
	CurveP521 CurveID = 25
	X25519    CurveID = 29

	x25519Kyber768Draft00 CurveID = 0x6399
)

// SignatureScheme 类型表示签名方案
type SignatureScheme uint16

// 定义各种签名方案
const (
	PKCS1WithSHA256 SignatureScheme = 0x0401
	PKCS1WithSHA384 SignatureScheme = 0x0501
	PKCS1WithSHA512 SignatureScheme = 0x0601

	PSSWithSHA256 SignatureScheme = 0x0804
	PSSWithSHA384 SignatureScheme = 0x0805
	PSSWithSHA512 SignatureScheme = 0x0806

	ECDSAWithP256AndSHA256 SignatureScheme = 0x0403
	ECDSAWithP384AndSHA384 SignatureScheme = 0x0503
	ECDSAWithP521AndSHA512 SignatureScheme = 0x0603

	Ed25519 SignatureScheme = 0x0807

	PKCS1WithSHA1 SignatureScheme = 0x0201
	ECDSAWithSHA1 SignatureScheme = 0x0203
)

// 定义状态类型为 OCSP
const (
	statusTypeOCSP uint8 = 1
)

// 定义各种 TLS 扩展的 ID
const (
	extensionServerName              uint16 = 0
	extensionStatusRequest           uint16 = 5
	extensionSupportedCurves         uint16 = 10
	extensionSupportedPoints         uint16 = 11
	extensionSignatureAlgorithms     uint16 = 13
	extensionALPN                    uint16 = 16
	extensionSCT                     uint16 = 18
	extensionExtendedMasterSecret    uint16 = 23
	extensionSessionTicket           uint16 = 35
	extensionPreSharedKey            uint16 = 41
	extensionEarlyData               uint16 = 42
	extensionSupportedVersions       uint16 = 43
	extensionCookie                  uint16 = 44
	extensionPSKModes                uint16 = 45
	extensionCertificateAuthorities  uint16 = 47
	extensionSignatureAlgorithmsCert uint16 = 50
	extensionKeyShare                uint16 = 51
	extensionQUICTransportParameters uint16 = 57
	extensionRenegotiationInfo       uint16 = 0xff01
	extensionECHOuterExtensions      uint16 = 0xfd00
	extensionEncryptedClientHello    uint16 = 0xfe0d
)

// readUint8LengthPrefixed 函数用于从 cryptobyte.String 中读取以 8 位长度前缀的数据
func readUint8LengthPrefixed(s *cryptobyte.String, out *[]byte) bool {
	return s.ReadUint8LengthPrefixed((*cryptobyte.String)(out))
}

// readUint16LengthPrefixed 函数用于从 cryptobyte.String 中读取以 16 位长度前缀的数据
func readUint16LengthPrefixed(s *cryptobyte.String, out *[]byte) bool {
	return s.ReadUint16LengthPrefixed((*cryptobyte.String)(out))
}

// readUint24LengthPrefixed 函数用于从 cryptobyte.String 中读取以 24 位长度前缀的数据
func readUint24LengthPrefixed(s *cryptobyte.String, out *[]byte) bool {
	return s.ReadUint24LengthPrefixed((*cryptobyte.String)(out))
}

// ClientHelloMsg 结构体表示 TLS 握手过程中的客户端问候消息
type ClientHelloMsg struct {
	Vers                             uint16            // TLS 版本
	Random                           []byte            // 随机数
	SessionId                        []byte            // 会话 ID
	CipherSuites                     []uint16          // 支持的密码套件
	CompressionMethods               []uint8           // 支持的压缩方法
	ServerName                       string            // 服务器名称
	OcspStapling                     bool              // 是否支持 OCSP 装订
	SupportedCurves                  []CurveID         // 支持的曲线
	SupportedPoints                  []uint8           // 支持的点格式
	TicketSupported                  bool              // 是否支持会话票据
	SessionTicket                    []uint8           // 会话票据
	SupportedSignatureAlgorithms     []SignatureScheme // 支持的签名算法
	SupportedSignatureAlgorithmsCert []SignatureScheme // 支持的证书签名算法
	SecureRenegotiationSupported     bool              // 是否支持安全重协商
	SecureRenegotiation              []byte            // 安全重协商数据
	ExtendedMasterSecret             bool              // 是否支持扩展主密钥
	AlpnProtocols                    []string          // 支持的 ALPN 协议
	Scts                             bool              // 是否支持 SCT
	SupportedVersions                []uint16          // 支持的 TLS 版本
	Cookie                           []byte            // Cookie
	KeyShares                        []keyShare        // 密钥共享信息
	EarlyData                        bool              // 是否支持早期数据
	PskModes                         []uint8           // PSK 模式
	PskIdentities                    []pskIdentity     // PSK 身份信息
	PskBinders                       [][]byte          // PSK 绑定器
	QuicTransportParameters          []byte            // QUIC 传输参数
	EncryptedClientHello             []byte            // 加密的客户端问候消息
}

// UnmarshalByByte 方法用于从字节切片中解析客户端问候消息
func (m *ClientHelloMsg) UnmarshalByByte(buf []byte) error {
	// 检查字节切片是否为空
	if len(buf) == 0 {
		return fmt.Errorf("len(buf) == 0")
	}
	// 检查字节切片的第一个字节是否为 0x16（TLS 握手消息类型）
	if buf[0] != 0x16 {
		return fmt.Errorf("!=0x16")
	}

	// 调用 unmarshal 方法解析剩余的数据
	return m.unmarshal(buf[5:])
}

// unmarshal 方法用于实际解析客户端问候消息
func (m *ClientHelloMsg) unmarshal(data []byte) error {
	// 初始化 ClientHelloMsg 结构体
	*m = ClientHelloMsg{}
	s := cryptobyte.String(data)

	// 跳过前 4 个字节
	if !s.Skip(4) ||
		// 读取 TLS 版本
		!s.ReadUint16(&m.Vers) ||
		// 读取随机数
		!s.ReadBytes(&m.Random, 32) ||
		// 读取会话 ID
		!readUint8LengthPrefixed(&s, &m.SessionId) {
		return fmt.Errorf("Skip error")
	}

	var cipherSuites cryptobyte.String
	// 读取支持的密码套件列表
	if !s.ReadUint16LengthPrefixed(&cipherSuites) {
		return fmt.Errorf("cipherSuites error")
	}

	m.CipherSuites = []uint16{}
	m.SecureRenegotiationSupported = false
	// 遍历密码套件列表
	for !cipherSuites.Empty() {
		var suite uint16
		// 读取单个密码套件
		if !cipherSuites.ReadUint16(&suite) {
			return fmt.Errorf("cipherSuites ReadUint16 error %d", suite)
		}
		// 检查是否支持安全重协商
		if suite == scsvRenegotiation {
			m.SecureRenegotiationSupported = true
		}
		m.CipherSuites = append(m.CipherSuites, suite)
	}

	// 读取支持的压缩方法
	if !readUint8LengthPrefixed(&s, &m.CompressionMethods) {
		return fmt.Errorf("CompressionMethods   error")
	}

	// 检查是否还有数据
	if s.Empty() {
		return fmt.Errorf("s  Empty  error")
	}

	var extensions cryptobyte.String
	// 读取扩展列表
	if !s.ReadUint16LengthPrefixed(&extensions) || !s.Empty() {
		return fmt.Errorf("ReadUint16LengthPrefixed  extensions error")
	}

	// 用于记录已经处理过的扩展
	seenExts := make(map[uint16]bool)
	// 遍历扩展列表
	for !extensions.Empty() {
		var extension uint16
		var extData cryptobyte.String
		// 读取扩展 ID 和扩展数据
		if !extensions.ReadUint16(&extension) ||
			!extensions.ReadUint16LengthPrefixed(&extData) {
			return fmt.Errorf("ReadUint16LengthPrefixed  extData error extension:%d  and  extData:%s", extension, extData)
		}

		// 检查该扩展是否已经处理过
		if seenExts[extension] {
			return fmt.Errorf("seenExts  extension error %d", extension)
		}

		seenExts[extension] = true
		switch extension {
		case extensionServerName:
			var nameList cryptobyte.String
			// 读取服务器名称列表
			if !extData.ReadUint16LengthPrefixed(&nameList) || nameList.Empty() {
				return fmt.Errorf("extensionServerName  nameList error %s", extData)
			}
			for !nameList.Empty() {
				var nameType uint8
				var serverName cryptobyte.String
				// 读取名称类型和服务器名称
				if !nameList.ReadUint8(&nameType) ||
					!nameList.ReadUint16LengthPrefixed(&serverName) ||
					serverName.Empty() {
					return fmt.Errorf("extensionServerName  nameList error %s", extData)
				}
				// 只处理类型为 0 的服务器名称
				if nameType != 0 {
					continue
				}
				// 确保只有一个服务器名称
				if len(m.ServerName) != 0 {
					return fmt.Errorf("extensionServerName  nameList error %s", extData)
				}
				m.ServerName = string(serverName)

				// 检查服务器名称是否以点结尾
				if strings.HasSuffix(m.ServerName, ".") {
					return fmt.Errorf("extensionServerName  nameList error %s", extData)
				}
			}
		case extensionStatusRequest:
			var statusType uint8
			var ignored cryptobyte.String
			// 读取状态类型和忽略的数据
			if !extData.ReadUint8(&statusType) ||
				!extData.ReadUint16LengthPrefixed(&ignored) ||
				!extData.ReadUint16LengthPrefixed(&ignored) {
				return fmt.Errorf("extensionStatusRequest     ")
			}
			// 判断是否支持 OCSP 装订
			m.OcspStapling = statusType == statusTypeOCSP
		case extensionSupportedCurves:
			var curves cryptobyte.String
			// 读取支持的曲线列表
			if !extData.ReadUint16LengthPrefixed(&curves) || curves.Empty() {
				return fmt.Errorf("extensionSupportedCurves     ")
			}
			for !curves.Empty() {
				var curve uint16
				// 读取单个曲线 ID
				if !curves.ReadUint16(&curve) {
					return fmt.Errorf("extensionSupportedCurves     ")
				}
				m.SupportedCurves = append(m.SupportedCurves, CurveID(curve))
			}
		case extensionSupportedPoints:
			// 读取支持的点格式
			if !readUint8LengthPrefixed(&extData, &m.SupportedPoints) ||
				len(m.SupportedPoints) == 0 {
				return fmt.Errorf("extensionSupportedPoints     ")
			}
		case extensionSessionTicket:
			// 标记支持会话票据
			m.TicketSupported = true
			// 读取会话票据数据
			extData.ReadBytes(&m.SessionTicket, len(extData))
		case extensionSignatureAlgorithms:
			var sigAndAlgs cryptobyte.String
			// 读取支持的签名算法列表
			if !extData.ReadUint16LengthPrefixed(&sigAndAlgs) || sigAndAlgs.Empty() {
				return fmt.Errorf("extensionSignatureAlgorithms     ")
			}
			for !sigAndAlgs.Empty() {
				var sigAndAlg uint16
				// 读取单个签名算法
				if !sigAndAlgs.ReadUint16(&sigAndAlg) {
					return fmt.Errorf("extensionSignatureAlgorithms     ")
				}
				m.SupportedSignatureAlgorithms = append(
					m.SupportedSignatureAlgorithms, SignatureScheme(sigAndAlg))
			}
		case extensionSignatureAlgorithmsCert:
			var sigAndAlgs cryptobyte.String
			// 读取支持的证书签名算法列表
			if !extData.ReadUint16LengthPrefixed(&sigAndAlgs) || sigAndAlgs.Empty() {
				return fmt.Errorf("extensionSignatureAlgorithmsCert     ")
			}
			for !sigAndAlgs.Empty() {
				var sigAndAlg uint16
				// 读取单个证书签名算法
				if !sigAndAlgs.ReadUint16(&sigAndAlg) {
					return fmt.Errorf("extensionSignatureAlgorithmsCert     ")
				}
				m.SupportedSignatureAlgorithmsCert = append(
					m.SupportedSignatureAlgorithmsCert, SignatureScheme(sigAndAlg))
			}
		case extensionRenegotiationInfo:
			// 读取安全重协商数据
			if !readUint8LengthPrefixed(&extData, &m.SecureRenegotiation) {
				return fmt.Errorf("extensionRenegotiationInfo     ")
			}
			// 标记支持安全重协商
			m.SecureRenegotiationSupported = true
		case extensionExtendedMasterSecret:
			// 标记支持扩展主密钥
			m.ExtendedMasterSecret = true
		case extensionALPN:
			var protoList cryptobyte.String
			// 读取支持的 ALPN 协议列表
			if !extData.ReadUint16LengthPrefixed(&protoList) || protoList.Empty() {
				return fmt.Errorf("extensionALPN     ")
			}
			for !protoList.Empty() {
				var proto cryptobyte.String
				// 读取单个 ALPN 协议
				if !protoList.ReadUint8LengthPrefixed(&proto) || proto.Empty() {
					return fmt.Errorf("extensionALPN     ")
				}
				m.AlpnProtocols = append(m.AlpnProtocols, string(proto))
			}
		case extensionSCT:
			// 标记支持 SCT
			m.Scts = true
		case extensionSupportedVersions:
			var versList cryptobyte.String
			// 读取支持的 TLS 版本列表
			if !extData.ReadUint8LengthPrefixed(&versList) || versList.Empty() {
				return fmt.Errorf("extensionSupportedVersions     ")
			}
			for !versList.Empty() {
				var vers uint16
				// 读取单个 TLS 版本
				if !versList.ReadUint16(&vers) {
					return fmt.Errorf("extensionSupportedVersions     ")
				}
				m.SupportedVersions = append(m.SupportedVersions, vers)
			}
		case extensionCookie:
			// 读取 Cookie 数据
			if !readUint16LengthPrefixed(&extData, &m.Cookie) ||
				len(m.Cookie) == 0 {
				return fmt.Errorf("extensionCookie     ")
			}
		case extensionKeyShare:
			var clientShares cryptobyte.String
			// 读取密钥共享信息列表
			if !extData.ReadUint16LengthPrefixed(&clientShares) {
				return fmt.Errorf("extensionKeyShare     ")
			}
			for !clientShares.Empty() {
				var ks keyShare
				// 读取单个密钥共享信息
				if !clientShares.ReadUint16((*uint16)(&ks.group)) ||
					!readUint16LengthPrefixed(&clientShares, &ks.data) ||
					len(ks.data) == 0 {
					return fmt.Errorf("extensionKeyShare     ")
				}
				m.KeyShares = append(m.KeyShares, ks)
			}
		case extensionEarlyData:
			// 标记支持早期数据
			m.EarlyData = true
		case extensionPSKModes:
			// 读取 PSK 模式
			if !readUint8LengthPrefixed(&extData, &m.PskModes) {
				return fmt.Errorf("extensionPSKModes     ")
			}
		case extensionQUICTransportParameters:
			// 读取 QUIC 传输参数
			m.QuicTransportParameters = make([]byte, len(extData))
			if !extData.CopyBytes(m.QuicTransportParameters) {
				return fmt.Errorf("extensionQUICTransportParameters     ")
			}
		case extensionPreSharedKey:
			// 确保扩展列表为空
			if !extensions.Empty() {
				return fmt.Errorf("extensionPreSharedKey     ")
			}
			var identities cryptobyte.String
			// 读取预共享密钥身份信息列表
			if !extData.ReadUint16LengthPrefixed(&identities) || identities.Empty() {
				return fmt.Errorf("extensionPreSharedKey     ")
			}
			for !identities.Empty() {
				var psk pskIdentity
				// 读取单个预共享密钥身份信息
				if !readUint16LengthPrefixed(&identities, &psk.label) ||
					!identities.ReadUint32(&psk.obfuscatedTicketAge) ||
					len(psk.label) == 0 {
					return fmt.Errorf("extensionPreSharedKey     ")
				}
				m.PskIdentities = append(m.PskIdentities, psk)
			}
			var binders cryptobyte.String
			// 读取预共享密钥绑定器列表
			if !extData.ReadUint16LengthPrefixed(&binders) || binders.Empty() {
				return fmt.Errorf("extensionPreSharedKey     ")
			}
			for !binders.Empty() {
				var binder []byte
				// 读取单个预共享密钥绑定器
				if !readUint8LengthPrefixed(&binders, &binder) ||
					len(binder) == 0 {
					return fmt.Errorf("extensionPreSharedKey     ")
				}
				m.PskBinders = append(m.PskBinders, binder)
			}
		default:
			continue
		}

		// 确保扩展数据已全部处理
		if !extData.Empty() {
			return fmt.Errorf(" !extData.Empty() ")
		}
	}

	return nil
}
