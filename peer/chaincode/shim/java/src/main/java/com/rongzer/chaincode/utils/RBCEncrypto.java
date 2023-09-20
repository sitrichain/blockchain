package com.rongzer.chaincode.utils;

import java.security.interfaces.ECPrivateKey;
import java.security.interfaces.ECPublicKey;

import org.bouncycastle.jce.provider.JCEECPublicKey;
import org.bouncycastle.math.ec.ECPoint;

/**
 * 加密对象
 * @author Administrator
 *
 */
public class RBCEncrypto {
	public ECPublicKey publicKey = null;
	public ECPrivateKey privateKey = null;
	private byte[] p2Buf = null;

	public RBCEncrypto(ECPublicKey publicKey,ECPrivateKey privateKey)
	{
		this.publicKey = publicKey;
		this.privateKey = privateKey;
		calP2();
	}
	

	private void calP2(){
		
		org.bouncycastle.jce.interfaces.ECPublicKey pub = (org.bouncycastle.jce.interfaces.ECPublicKey)publicKey;
		ECPoint p2 = null;
		if (publicKey instanceof JCEECPublicKey){
			p2 = ((JCEECPublicKey)publicKey).engineGetQ().multiply(privateKey.getS());
		}else {
			 p2 = pub.getQ().multiply(privateKey.getS());
		}
				
		 
		p2Buf = p2.getEncoded(false);
	}

	
	/**
	 * 加密
	 * @param msg
	 * @return
	 */
	public byte[] getCryptogram(String modelName,String tableName,String colName,String customerNo){
		String addStr = modelName+"-"+tableName+"-"+colName;
		if (StringUtil.isNotEmpty(customerNo)){
			addStr +="-"+customerNo;
		}
		byte[] p2 = ChainCodeUtils.hash(p2Buf,addStr.getBytes());
		return p2;
	}

	
	
}
