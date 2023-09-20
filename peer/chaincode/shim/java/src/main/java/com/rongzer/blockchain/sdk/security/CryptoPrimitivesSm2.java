package com.rongzer.blockchain.sdk.security;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.StringWriter;
import java.math.BigInteger;
import java.security.KeyPair;
import java.security.PrivateKey;
import java.security.cert.Certificate;
import java.security.interfaces.ECPrivateKey;
import java.security.interfaces.ECPublicKey;
import java.util.Collection;
import java.util.Properties;

import javax.security.auth.x500.X500Principal;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.bouncycastle.asn1.ASN1InputStream;
import org.bouncycastle.asn1.ASN1ObjectIdentifier;
import org.bouncycastle.asn1.ASN1Sequence;
import org.bouncycastle.asn1.DERBitString;
import org.bouncycastle.asn1.DEROctetString;
import org.bouncycastle.asn1.DERSequenceGenerator;
import org.bouncycastle.asn1.DERTaggedObject;
import org.bouncycastle.asn1.DLSequence;
import org.bouncycastle.asn1.x500.X500Name;
import org.bouncycastle.asn1.x509.AlgorithmIdentifier;
import org.bouncycastle.asn1.x509.SubjectPublicKeyInfo;
import org.bouncycastle.asn1.x509.X509CertificateStructure;
import org.bouncycastle.crypto.AsymmetricCipherKeyPair;
import org.bouncycastle.crypto.CryptoException;
import org.bouncycastle.crypto.generators.ECKeyPairGenerator;
import org.bouncycastle.crypto.params.ECPrivateKeyParameters;
import org.bouncycastle.crypto.params.ECPublicKeyParameters;
import org.bouncycastle.jce.provider.JCEECPrivateKey;
import org.bouncycastle.jce.provider.JCEECPublicKey;
import org.bouncycastle.math.ec.ECPoint;
import org.bouncycastle.openssl.jcajce.JcaPEMWriter;
import org.bouncycastle.operator.ContentSigner;
import org.bouncycastle.operator.OperatorCreationException;
import org.bouncycastle.pkcs.PKCS10CertificationRequest;
import org.bouncycastle.pkcs.PKCS10CertificationRequestBuilder;
import org.apache.xerces.impl.dv.util.Base64;
import org.bouncycastle.util.encoders.Hex;
import org.bouncycastle.util.io.pem.PemObject;
import com.rongzer.blockchain.sdk.exception.InvalidArgumentException;

import com.rongzer.chaincode.utils.Hash;
import com.security.cipher.sm.SM2;
import com.security.cipher.sm.SM2Utils;
import com.security.cipher.sm.SM3Digest;


public class CryptoPrimitivesSm2 implements CryptoSuite {
	
    private static final Log logger = LogFactory.getLog(CryptoPrimitivesSm2.class);

	static final SM2 sm2 = SM2.Instance();

	@Override
	public void init() throws CryptoException, InvalidArgumentException {
		
	}

	@Override
	public void setProperties(Properties properties) throws CryptoException,
			InvalidArgumentException {
		
	}

	@Override
	public Properties getProperties() {
        Properties properties = new Properties();
        return properties;
	}


	@Override
	public Certificate bytesToCertificate(byte[] certBytes)
			throws CryptoException {
		return null;
	}
	
	@Override
	public ECPublicKey bytesToPublicKey(byte[] pubBuf)
			throws CryptoException {
		if (pubBuf == null || pubBuf.length<65){
			throw(new CryptoException("pub buf length is error"));
		}
		if (pubBuf.length>65){			
			pubBuf = getCSPK(pubBuf);
		}
		
		ECPoint ecPoint1 = sm2.ecc_curve.decodePoint(pubBuf);
		
		ECPublicKeyParameters pubParams = new ECPublicKeyParameters(ecPoint1,sm2.ecc_bc_spec);
		
        ECPublicKey publicKey = new JCEECPublicKey("SM2WithSM3",pubParams);
		return publicKey;
	}
	
    public ECPrivateKey bytesToPrivateKey(byte[] priBytes) throws CryptoException {
		ECPrivateKeyParameters priParams = new ECPrivateKeyParameters(new BigInteger(priBytes), sm2.ecc_bc_spec);
        ECPrivateKey privateKey = new JCEECPrivateKey("SM2WithSM3",priParams);
        return privateKey;
    }

	@Override
	public PKCS10CertificationRequest generateCertificationRequest(
			String subject, KeyPair pair) throws OperatorCreationException {
		ASN1ObjectIdentifier SM32_oid = new ASN1ObjectIdentifier("1.2.840.10045.2.1");
		ASN1ObjectIdentifier SM32_pid = new ASN1ObjectIdentifier("1.2.156.10197.1.301");
		byte[] buf = pair.getPublic().getEncoded();
		if (pair.getPublic() instanceof JCEECPublicKey)
		{
			buf = ((JCEECPublicKey)pair.getPublic()).getQ().getEncoded(false);            
		}
		
		logger.debug("SM2 公钥=" + pair.getPublic());

		AlgorithmIdentifier alg = new AlgorithmIdentifier(SM32_oid,SM32_pid); 
		SubjectPublicKeyInfo publicKeyInfo= new SubjectPublicKeyInfo(alg,buf) ;
		
		X500Name subject1 = X500Name.getInstance(( new X500Principal("CN=" + subject)).getEncoded()) ;
        PKCS10CertificationRequestBuilder p10Builder = new PKCS10CertificationRequestBuilder(subject1, publicKeyInfo);

        ContentSigner contentSigner = buildContentSigner("",pair.getPrivate());

        return p10Builder.build(contentSigner);
	}
	
	
	private ContentSigner buildContentSigner(String userId,PrivateKey privateKey){
		
		return new ContentSigner(){
			
			private CryptoPrimitivesSm2.SM2SignatureOutputStream stream;
			
			
			public AlgorithmIdentifier getAlgorithmIdentifier() {
				ASN1ObjectIdentifier SM32_oid = new ASN1ObjectIdentifier("1.2.840.10045.4.3.2");
				ASN1ObjectIdentifier SM32_pid = new ASN1ObjectIdentifier("1.2.156.10197.1.301");

				return new AlgorithmIdentifier(SM32_oid,SM32_pid); 

			}

			public OutputStream getOutputStream() {
				this.stream = new SM2SignatureOutputStream(userId,privateKey);
				return stream;
			}

			public byte[] getSignature() {
				return stream.bSign;
			}
			
		};
	}
	
	  private class SM2SignatureOutputStream extends OutputStream {
		  
		  String userId = "";
		  PrivateKey privateKey = null;
		  
		  byte[] bSign = null;
		  SM2SignatureOutputStream(String userId,PrivateKey privateKey){
			  this.userId = userId;
			  this.privateKey = privateKey;
		  }
		  
		  public void write(byte[] bytes, int off, int len) throws IOException {

	      }

	      public void write(byte[] bytes) throws IOException {
	    	  
	    	  logger.debug("cert sign hash:"+Hex.toHexString(Hash.hash(bytes)));
	    	  bSign = SM2Utils.sign(userId.getBytes(),((ECPrivateKey)privateKey).getS().toByteArray(), Hash.hash(bytes));
	    	  logger.debug("sign result:"+Hex.toHexString(bSign));
	  		
	      }

	      public void write(int b) throws IOException {

	      }


	  }

	@Override
	public String certificationRequestToPEM(PKCS10CertificationRequest csr)
			throws IOException {
        PemObject pemCSR = new PemObject("CERTIFICATE REQUEST", csr.getEncoded());

        StringWriter str = new StringWriter();
        JcaPEMWriter pemWriter = new JcaPEMWriter(str);
        pemWriter.writeObject(pemCSR);
        pemWriter.close();
        str.close();
        return str.toString();
	}
	
	@Override
	public void loadCACertificates(Collection<Certificate> certificates)
			throws CryptoException {
		
	}

	@Override
	public void loadCACertificatesAsBytes(Collection<byte[]> certificates)
			throws CryptoException {
		
	}

	@Override
	public KeyPair keyGen() throws CryptoException {
		ECKeyPairGenerator generator = sm2.ecc_key_pair_generator;
        AsymmetricCipherKeyPair keypair = generator.generateKeyPair();
        ECPrivateKeyParameters privParams = (ECPrivateKeyParameters) keypair.getPrivate();
        ECPublicKeyParameters pubParams = (ECPublicKeyParameters) keypair.getPublic();
       
        JCEECPrivateKey privateKey = new JCEECPrivateKey("SM2WithSM3",privParams);
        JCEECPublicKey publicKey = new JCEECPublicKey("SM2WithSM3",pubParams);
        logger.debug("SM2 Keygen Pub:"+publicKey.toString());
        KeyPair keypair1 = new KeyPair(publicKey,privateKey);
		return keypair1;
	}

	@Override
	public byte[] sign(PrivateKey key,byte[] userId, byte[] plainText) throws CryptoException {
		
		byte[] bReturn = null;
		try
		{			
			byte[] digest = Hash.hash(plainText);
			if (userId == null){
				userId = new byte[0];
			}

			bReturn = SM2Utils.sign(userId, ((ECPrivateKey)key).getS().toByteArray(), digest);
			if (userId != null&& userId.length>0){
				//处理返回结时,增加userId的返回
			    ByteArrayInputStream bis = new ByteArrayInputStream(bReturn);
			    ASN1InputStream dis = new ASN1InputStream(bis);
			    DLSequence seq = (DLSequence) dis.readObject();
			    dis.close();
			    bis.close();
			    
			    ByteArrayOutputStream baos = new ByteArrayOutputStream();
			    DERSequenceGenerator seq1 = new DERSequenceGenerator(baos);
		
			    seq1.addObject(seq.getObjectAt(0));
			    seq1.addObject(seq.getObjectAt(1));
			    seq1.addObject(new DERTaggedObject(0,new DEROctetString(userId)));
			    seq1.close();
			    baos.close();
			    return baos.toByteArray();
			}

		
		}catch(Exception e)
		{
			throw new CryptoException(e.getMessage());
		}
		
		return bReturn;
	}

	@Override
	public boolean verify(byte[] certificate, String signatureAlgorithm,
			byte[] signature, byte[] plainText) throws CryptoException {

		boolean bReturn = false;
		try{
			byte[] digest = Hash.hash(plainText);
			
			byte[] userId = new byte[0];
				//处理返回结时,增加userId的返回
			    ByteArrayInputStream bis = new ByteArrayInputStream(signature);
			    ASN1InputStream dis = new ASN1InputStream(bis);
			    DLSequence seq = (DLSequence) dis.readObject();
			    dis.close();
			    bis.close();
			   
			if (seq.size() >2) {
				userId = ((DEROctetString)((DERTaggedObject)seq.getObjectAt(2)).getObject()).getOctets();
			}
			
			bReturn = SM2Utils.verifySign(userId, getCSPK(certificate), digest,signature);
		}catch(Exception e)
		{
			throw new CryptoException(e.getMessage());
		}
		
		return bReturn;
	}

	@Override
	public byte[] hash(byte[] plainText) {
        SM3Digest sm3 = new SM3Digest();
        sm3.update(plainText, 0, plainText.length);

        byte[] md = new byte[32];
        sm3.doFinal(md, 0);
        return md;
	}

	/**
	 * 从证书获取公钥串
	 * @param csCert
	 * @return
	 */
	private byte[] getCSPK(byte[] csCert)
    {
	 

	 String strCert = new String(csCert);
	 strCert = strCert.replaceAll("-----BEGIN -----", "");
	 strCert = strCert.replaceAll("-----END -----", "");

	 strCert = strCert.replaceAll("-----BEGIN CERTIFICATE-----", "");
	 strCert = strCert.replaceAll("-----END CERTIFICATE-----", "");
	 strCert = strCert.replaceAll("\r", "");
	 strCert = strCert.replaceAll("\n", "");

	 ASN1Sequence seq = null;
		ASN1InputStream aIn;
		try {
			InputStream inStream = new ByteArrayInputStream(Base64.decode(strCert));

			aIn = new ASN1InputStream(inStream);
			seq = (ASN1Sequence) aIn.readObject();
			aIn.close();
			X509CertificateStructure cert = new X509CertificateStructure(seq);
			SubjectPublicKeyInfo subjectPublicKeyInfo = cert
					.getSubjectPublicKeyInfo();

            DERBitString publicKeyData = subjectPublicKeyInfo.getPublicKeyData();

            byte[] publicKey = publicKeyData.getEncoded();
            byte[] encodedPublicKey = publicKey;
            byte[] eP = new byte[65];
            eP[0] = 0x04;
            System.arraycopy(encodedPublicKey, 4, eP, 1, eP.length - 1);
            return eP;
        }
        catch (Exception e)
        {
            e.printStackTrace();
        }
        return null;
    }

}
