package com.security.cipher.sm;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.math.BigInteger;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.bouncycastle.asn1.ASN1InputStream;
import org.bouncycastle.asn1.ASN1Integer;
import org.bouncycastle.asn1.DERBitString;
import org.bouncycastle.asn1.DEROctetString;
import org.bouncycastle.asn1.DERSequenceGenerator;
import org.bouncycastle.asn1.DLSequence;
import org.bouncycastle.crypto.AsymmetricCipherKeyPair;
import org.bouncycastle.crypto.params.ECPrivateKeyParameters;
import org.bouncycastle.crypto.params.ECPublicKeyParameters;
import org.bouncycastle.math.ec.ECPoint;
import org.bouncycastle.math.ec.WNafL2RMultiplier;
import org.bouncycastle.pqc.math.linearalgebra.ByteUtils;
import org.bouncycastle.util.BigIntegers;
import org.apache.xerces.impl.dv.util.Base64;
import org.bouncycastle.util.encoders.Hex;

public class SM2Utils 
{
    private static final Log logger = LogFactory.getLog(SM2Utils.class);

	public static byte[] encrypt(byte[] publicKey, byte[] data) throws IOException
	{
		if (publicKey == null || publicKey.length == 0)
		{
			return null;
		}

		if (data == null || data.length == 0)
		{
			return null;
		}
		
		byte[] source = new byte[data.length];
		System.arraycopy(data, 0, source, 0, data.length);
		
		Cipher cipher = new Cipher();
		SM2 sm2 = SM2.Instance();
		ECPoint userKey = sm2.ecc_curve.decodePoint(publicKey);
		
		ECPoint c1 = cipher.Init_enc(sm2, userKey);
		cipher.Encrypt(source);
		byte[] c3 = new byte[32];
		cipher.Dofinal(c3);		
		//DERInteger x = new DERInteger(c1.getX().toBigInteger());
		//DERInteger y = new DERInteger(c1.getY().toBigInteger());
		
		byte[] bC1 = c1.getEncoded(false);
		byte[] bOut = new byte[bC1.length+source.length+c3.length];
		
		System.arraycopy(bC1, 0, bOut, 0, bC1.length);
		System.arraycopy(source, 0, bOut, bC1.length, source.length);
		System.arraycopy(c3, 0, bOut, bC1.length + source.length, c3.length);
		
		return bOut;
	}
	
	public static byte[] decrypt(byte[] privateKey, byte[] encryptedData) throws IOException
	{
		if (privateKey == null || privateKey.length == 0)
		{
			return null;
		}
		
		if (encryptedData == null || encryptedData.length == 0)
		{
			return null;
		}
		
		byte[] enc = new byte[encryptedData.length];
		System.arraycopy(encryptedData, 0, enc, 0, encryptedData.length);
		
		SM2 sm2 = SM2.Instance();
		BigInteger userD = new BigInteger(1, privateKey);
						
		byte[] bC1 = new byte[65];
		System.arraycopy(encryptedData, 0, bC1, 0, bC1.length);

		ECPoint c1 = sm2.ecc_curve.decodePoint(bC1).normalize();
		
		byte[] c3 = new byte[32];
		byte[] c2 = new byte[encryptedData.length - 97];
		System.arraycopy(encryptedData, encryptedData.length-c3.length, c3, 0, c3.length);
		System.arraycopy(encryptedData, encryptedData.length-c3.length-c2.length, c2, 0, c2.length);
		
		Cipher cipher = new Cipher();
		cipher.Init_dec(userD, c1);
		cipher.Decrypt(c2);
		cipher.Dofinal(c3);
		return c2;
	}
	
	public static byte[] sign(byte[] userId, byte[] privateKey, byte[] sourceData) throws IOException
	{
		if (privateKey == null || privateKey.length == 0)
		{
			return null;
		}
		
		if (sourceData == null || sourceData.length == 0)
		{
			return null;
		}
		
		SM2 sm2 = SM2.Instance();
		BigInteger userD = new BigInteger(privateKey);		
		ECPoint userKey = sm2.ecc_point_g.multiply(userD);
		
		SM3Digest sm3 = new SM3Digest();
		byte[] z = sm2.sm2GetZ(userId, userKey);
	    
		sm3.update(z, 0, z.length);
		//不做hash处理
		byte[] sourceHash = sourceData;
		sm3.update(sourceHash, 0, sourceHash.length);
	    byte[] md = new byte[32];
	    sm3.doFinal(md, 0);
	    
	    logger.debug("SM3 Digest摘要值: " + Util.getHexString(md));
	    
	    SM2Result sm2Result = new SM2Result();
	    sm2.sm2Sign(md, userD, userKey, sm2Result);
	    logger.debug("r: " + sm2Result.r.toString(16));
	    logger.debug("s: " + sm2Result.s.toString(16));
	    
	    ByteArrayOutputStream baos = new ByteArrayOutputStream();
	    DERSequenceGenerator seq = new DERSequenceGenerator(baos);
	    
	    ASN1Integer d_r = new ASN1Integer(sm2Result.r);
	    ASN1Integer d_s = new ASN1Integer(sm2Result.s);

	   /* 
	    DERInteger d_r = new DERInteger(sm2Result.r);
	    DERInteger d_s = new DERInteger(sm2Result.s);*/
	    seq.addObject(d_r);
	    seq.addObject(d_s);
	    seq.close();
	    
	    return baos.toByteArray();
	    /*
	    DERObject sign = new DERSequence(v2);
	    byte[] signdata = sign.getDEREncoded();
		return signdata;*/
	}
	
	public static boolean verifySign(byte[] userId, byte[] publicKey, byte[] sourceData, byte[] signData) throws IOException
	{
		if (publicKey == null || publicKey.length == 0)
		{
			return false;
		}
		
		if (sourceData == null || sourceData.length == 0)
		{
			return false;
		}
		
		SM2 sm2 = SM2.Instance();
		ECPoint userKey = sm2.ecc_curve.decodePoint(publicKey);

		SM3Digest sm3 = new SM3Digest();
		byte[] z = sm2.sm2GetZ(userId, userKey);
		sm3.update(z, 0, z.length);
		
		//不做hash处理
		//byte[] sourceHash = getMsgHash(sourceData);
		
		byte[] sourceHash = sourceData;
		sm3.update(sourceHash, 0, sourceHash.length);
	    byte[] md = new byte[32];
	    sm3.doFinal(md, 0);
	    logger.debug("SM3摘要值: " + Util.getHexString(md));
		
	    ByteArrayInputStream bis = new ByteArrayInputStream(signData);
	    ASN1InputStream dis = new ASN1InputStream(bis);
	    DLSequence seq = (DLSequence) dis.readObject();
	    dis.close();
        BigInteger r = ((ASN1Integer) seq.getObjectAt(0)).getPositiveValue();
        BigInteger s = ((ASN1Integer) seq.getObjectAt(1)).getPositiveValue();
        
	   
	    SM2Result sm2Result = new SM2Result();
	    sm2Result.r = r;
	    sm2Result.s = s;
	    logger.debug("r: " + sm2Result.r.toString(16));
	    logger.debug("s: " + sm2Result.s.toString(16));	    
	    
	    sm2.sm2Verify(md, userKey, sm2Result.r, sm2Result.s, sm2Result);
        return sm2Result.r.equals(sm2Result.R);
	}
	
    public static byte[] getMsgHash(byte[] sourceData)
    {
        SM3Digest sm3 = new SM3Digest();
        sm3.update(sourceData, 0, sourceData.length);

        byte[] md = new byte[32];
        sm3.doFinal(md, 0);
        return md;

    }
    
	public static void main(String[] args) throws Exception 
	{
		SM2Test1();
	}
	
	
	
	public static void SM2Test() throws Exception 
	{
		byte[] m = new byte[1];
		byte[] p2 = new byte[1];
		byte[] p3 = new byte[1];
		byte[] p4 = new byte[1];
		m[0] = 'k';
		
		p2[0] = 'c';
		p3[0] = 'd';
		p4[0] = 'c';
		
		System.out.println(new String(m));
		
		m[0] ^= p2[0];
		
		System.out.println(new String(m));


		m[0] ^= p3[0];
		p4[0] ^= p3[0];
		m[0] ^= p4[0];

		System.out.println(new String(m));

		String plainText = "加密测";
		for (int i=0;i<4;i++)
		{
			plainText += plainText;
		}
		byte[] sourceData = plainText.getBytes();
		/*byte[] sourceData = new byte[32];
		for (int i=0;i<32;i++)
		{
			sourceData[i] =0x00;
		}*/
		System.out.println("数据长度："+sourceData.length);
		//byte[] bT = Util.hexStringToBytes("FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFC");
		// 国密规范测试私钥
		String prik = "128B2FA8BD433C6C068C8D803DFF79792A519A55171B1B650C23661D15897263";
		String pubk = "040AE4C7798AA0F119471BEE11825BE46202BB79E2A5844495E97C04FF4DF2548A7C0240F88F1CD4E16352A73C17B7F16F07353E53A176D684A9FE0C6BB798E857";
					   
		prik = "02ceef5444128b8cbc0c7ef8d5910ca98cfad73d790855c44ff99e9d863cf82a";
		pubk = "0414a629e872b78872941117562cccc1f7f1660b77c7ee1ac8d1dc03a6891469d26657155c050e616fcbc209b633415cf699e88359ba0233619d82ceaf771b1a60";
		
		
		String prikS = new String(Base64.encode(Util.hexToByte(prik)));
		// 国密规范测试公钥
		String pubkS = new String(Base64.encode(Util.hexToByte(pubk)));
		System.out.println("prik: " + prik);
		System.out.println("prikS: " + prikS);
		System.out.println("pubk: " + pubk);		
		System.out.println("pubkS: " + pubkS);
		System.out.println("");
		System.out.println("");
		
		
		// 国密规范测试用户ID
		//String userId = "ALICE123@YAHOO.COM";
		String userId = "";

		System.out.println("ID: " + Util.getHexString(userId.getBytes()));
		System.out.println("");
		
		System.out.println("签名: ");
		byte[] c = SM2Utils.sign(userId.getBytes(), Base64.decode(prikS), sourceData);
		System.out.println("sign: " + Util.getHexString(c));
		System.out.println("");
		
		
		System.out.println("验签: ");
		/*
		byte[] signByte = Util.hexStringToBytes("08080B3E097E7CB30CDC4A9DD7550257A1346919A5799FCC9BBBDA950F322D5CA7278955AA691EE10D40F6F8D52F934937D7F151132D59ABB30BD508E72867B5");
	    ByteArrayOutputStream baos = new ByteArrayOutputStream();
	    DERSequenceGenerator seq = new DERSequenceGenerator(baos);
	    BigInteger r = BigIntegers.fromUnsignedByteArray(signByte, 0, 32);
	    BigInteger s = BigIntegers.fromUnsignedByteArray(signByte, 32, 32);

	    ASN1Integer d_r = new ASN1Integer(r );
	    ASN1Integer d_s = new ASN1Integer(s);
	    seq.addObject(d_r);
	    seq.addObject(d_s);
	    seq.close();
	    
		boolean vs = SM2Utils.verifySign(userId.getBytes(), Base64.decode(pubkS.getBytes()), sourceData, baos.toByteArray());
	    */
	    boolean vs = SM2Utils.verifySign(userId.getBytes(), Base64.decode(pubkS), sourceData, c);
		System.out.println("验签结果: " + vs);
		System.out.println("");


		System.out.println("加密: ");
		byte[] cipherText = SM2Utils.encrypt(Base64.decode(pubkS), sourceData);
		System.out.println(ByteUtils.toHexString(cipherText));

		System.out.println(new String(Base64.encode(cipherText)));
		System.out.println("");
		
		System.out.println("解密: ");
		plainText = new String(SM2Utils.decrypt(Base64.decode(prikS), cipherText));
		System.out.println(plainText);
		
	}
	
	public static void SM2Test1() throws Exception 
	{
		
		SM2 sm2 = SM2.Instance();
		WNafL2RMultiplier mul = (WNafL2RMultiplier)sm2.ecc_curve.getMultiplier();
		  int coord = sm2.ecc_curve.getCoordinateSystem();

		AsymmetricCipherKeyPair key1 = sm2.ecc_key_pair_generator.generateKeyPair();
		ECPrivateKeyParameters ecpriv1 = (ECPrivateKeyParameters) key1.getPrivate();
		ECPublicKeyParameters ecpub1 = (ECPublicKeyParameters) key1.getPublic();
		BigInteger k1 = ecpriv1.getD();
		ECPoint K1 = ecpub1.getQ();
		System.out.println("k1:"+Hex.toHexString(k1.toByteArray()));
		System.out.println("K1:"+Hex.toHexString(K1.getEncoded()));
		
		AsymmetricCipherKeyPair key2 = sm2.ecc_key_pair_generator.generateKeyPair();
		ECPrivateKeyParameters ecpriv2 = (ECPrivateKeyParameters) key2.getPrivate();
		ECPublicKeyParameters ecpub2 = (ECPublicKeyParameters) key2.getPublic();
		BigInteger k2 = ecpriv2.getD();
		ECPoint K2 = ecpub2.getQ();
		System.out.println("k2:"+Hex.toHexString(k2.toByteArray()));
		System.out.println("K2:"+Hex.toHexString(K2.getEncoded()));
		
		AsymmetricCipherKeyPair key3 = sm2.ecc_key_pair_generator.generateKeyPair();
		ECPrivateKeyParameters ecpriv3 = (ECPrivateKeyParameters) key3.getPrivate();
		ECPublicKeyParameters ecpub3 = (ECPublicKeyParameters) key3.getPublic();
		BigInteger k3 = ecpriv3.getD();
		ECPoint K3 = ecpub3.getQ();
		System.out.println("k3:"+Hex.toHexString(k3.toByteArray()));
		System.out.println("K3:"+Hex.toHexString(K3.getEncoded()));
		
		ECPoint R1 = K1.add(K2).multiply(k3);
		ECPoint R2 = K3.multiply(k1).add(K3.multiply(k2));
		ECPoint R3 =  K3.multiply(k1.add(k2));
		System.out.println("R1:"+Hex.toHexString(R1.getEncoded()));
		System.out.println("R2:"+Hex.toHexString(R2.getEncoded()));
		System.out.println("R3:"+Hex.toHexString(R3.getEncoded()));
		
		
		String colName = "col1";
		byte[] msg = "测试字符串".getBytes();
		
		

	}
	
	
	public static void opensslTest() throws Exception 
	{
		String priPem = "MHcCAQEEIAThNIceWlkimoVIHuJcqzStF6LBNWvZKuTB5nvBeCfboAoGCCqBHM9VAYItoUQDQgAE4UDrEoR51zVRLaDi28tKliNxMQI04MyH/CEp88en1hYR1mWxgXIuMtwxp6ZxBbRSl8a7F5nr/3S3qrmb8aL0nA==";
		
		byte[] bPri = Base64.decode(priPem);
		
		String pubPem = "MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAE4UDrEoR51zVRLaDi28tKliNxMQI04MyH/CEp88en1hYR1mWxgXIuMtwxp6ZxBbRSl8a7F5nr/3S3qrmb8aL0nA==";
		byte[] bPub = Base64.decode(pubPem);

		
		ByteArrayInputStream bis = new ByteArrayInputStream(bPri);
	    ASN1InputStream dis = new ASN1InputStream(bis);
	    DLSequence seqPri = (DLSequence) dis.readObject();
	    DEROctetString spri = (DEROctetString) seqPri.getObjectAt(1);
	    dis.close();
	    bis = new ByteArrayInputStream(bPub);
	    dis = new ASN1InputStream(bis);
	    DLSequence seqPub = (DLSequence) dis.readObject();
	    
	    DERBitString dPub = (DERBitString)seqPub.getObjectAt(1);
	    dis.close();
		System.out.println("priPem: " + seqPri);
		System.out.println("pubPem: " + seqPub);

		
	    
		String plainText = "111";
		byte[] sourceData = plainText.getBytes();
		
		// 国密规范测试私钥
		String prik = Util.byteToHex(spri.getOctets());
		String pubk =  Util.byteToHex(dPub.getOctets());
					   

		System.out.println("prik: " + prik);
		System.out.println("pubk: " + pubk);		
		System.out.println("");
		System.out.println("");
		
		
		// 国密规范测试用户ID
		//String userId = "ALICE123@YAHOO.COM";
		String userId = "";

		System.out.println("ID: " + Util.getHexString(userId.getBytes()));
		System.out.println("");
		
		System.out.println("签名: ");
		byte[] c = SM2Utils.sign(userId.getBytes(),Util.hexStringToBytes(prik), sourceData);
		System.out.println("sign: " + Util.getHexString(c));
		System.out.println("");
		
		
		System.out.println("验签: ");

	    boolean vs = SM2Utils.verifySign(userId.getBytes(), Util.hexStringToBytes(pubk), sourceData, c);
		System.out.println("验签结果: " + vs);
		System.out.println("");


		System.out.println("加密: ");
		byte[] cipherText = SM2Utils.encrypt(Util.hexStringToBytes(pubk), sourceData);
		System.out.println(new String(Base64.encode(cipherText)));
		System.out.println("");
		
		System.out.println("解密: ");
		plainText = new String(SM2Utils.decrypt(Util.hexStringToBytes(prik), cipherText));
		System.out.println(plainText);
	}
	
	
	
	public static void KEYtest() throws Exception 
	{
		String plainText = "abc";
		byte[] sourceData = plainText.getBytes();
System.out.println("msg Hash:"+Util.byteToHex(getMsgHash(sourceData)));
		// 国密规范测试私钥
		String pubk = "041D919328AE326067C1D4B175241D3B0C58D9C272DA04261DCA0E590E712E07EBC17A9E43552F4B084B5077653F078045AC9FEB42DAEF16E5E347359CEF0DB8CC";
					   		
		
		// 国密规范测试公钥
		String pubkS = new String(Base64.encode(Util.hexToByte(pubk)));
		System.out.println("pubk: " + pubk);		
		System.out.println("pubkS: " + pubkS);
		
		// 国密规范测试用户ID
		//String userId = "ALICE123@YAHOO.COM";
		String userId = "";

		System.out.println("ID: " + Util.getHexString(userId.getBytes()));
		System.out.println("");

		
		System.out.println("验签: ");
		//byte[] signByte = Util.hexStringToBytes("5DFC30D85CF0040C425BB3CF8B0DEBF6EF27E792FE9BBC8C1133CA6C00AC5C6159FE09CD14FDDB622A5CEBB2A1D68414E4AA86FC3F773F1DE8116E0478442EC5");
		byte[] signByte = Util.hexStringToBytes("099A07C38C896414FED8966EB0BF5FF5C510181A42BE9305922BCE51BE677A6D6DEA6589417BC9D598AF8818020CA70625A5BAFAE6ADAB85FBC003AE1ABAFDCB");
	    ByteArrayOutputStream baos = new ByteArrayOutputStream();
	    DERSequenceGenerator seq = new DERSequenceGenerator(baos);
	    BigInteger r = BigIntegers.fromUnsignedByteArray(signByte, 0, 32);
	    BigInteger s = BigIntegers.fromUnsignedByteArray(signByte, 32, 32);

	    ASN1Integer d_r = new ASN1Integer(r );
	    ASN1Integer d_s = new ASN1Integer(s);
	    seq.addObject(d_r);
	    seq.addObject(d_s);
	    seq.close();
	    
		boolean vs = SM2Utils.verifySign(userId.getBytes(), Util.hexToByte(pubk), sourceData, baos.toByteArray());
	  
		System.out.println("验签结果: " + vs);
		System.out.println("");

	}
	
	
}
