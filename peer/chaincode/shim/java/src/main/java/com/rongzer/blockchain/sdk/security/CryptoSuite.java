/*
 *  Copyright 2016,2017 DTCC, Fujitsu Australia Software Technology, IBM - All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */
package com.rongzer.blockchain.sdk.security;

import java.io.IOException;
import java.security.KeyPair;
import java.security.PrivateKey;
import java.security.cert.Certificate;
import java.security.interfaces.ECPrivateKey;
import java.security.interfaces.ECPublicKey;
import java.util.Collection;
import java.util.Properties;

import org.bouncycastle.crypto.CryptoException;
import org.bouncycastle.operator.OperatorCreationException;
import org.bouncycastle.pkcs.PKCS10CertificationRequest;
import com.rongzer.blockchain.sdk.exception.InvalidArgumentException;

import com.rongzer.chaincode.utils.ChainCodeUtils;

/**
 * All packages for PKI key creation/signing/verification implement this interface
 */
public interface CryptoSuite {
    /**
     * implementation specific initialization. Whoever constructs a CryptoSuite instance <b>MUST</b> call
     * init before using the instance
     *
     * @throws CryptoException
     * @throws InvalidArgumentException
     */
    void init() throws CryptoException, InvalidArgumentException;

    /**
     * Pass in implementation specific properties to the CryptoSuite
     *
     * @param properties A {@link java.util.Properties} object. The key/value pairs are implementation specific
     * @throws CryptoException
     * @throws InvalidArgumentException
     */
    void setProperties(Properties properties) throws CryptoException, InvalidArgumentException;

    /**
     * @return the {@link Properties} object containing implementation specific key generation properties
     */
    Properties getProperties();

    /**
     * Set the Certificate Authority certificates to be used when validating a certificate chain of trust
     *
     * @param certificates A collection of {@link java.security.cert.Certificate}s
     * @throws CryptoException
     */
    void loadCACertificates(Collection<Certificate> certificates) throws CryptoException;

    /**
     * Set the Certificate Authority certificates to be used when validating a certificate chain of trust.
     *
     * @param certificates a collection of certificates in PEM format
     * @throws CryptoException
     */
    void loadCACertificatesAsBytes(Collection<byte[]> certificates) throws CryptoException;

    /**
     * Generate a key.
     *
     * @return the generated key.
     * @throws CryptoException
     */
    KeyPair keyGen() throws CryptoException;

    /**
     * Sign the specified byte string.
     *
     * @param key       the {@link java.security.PrivateKey} to be used for signing
     * @param plainText the byte string to sign
     * @return the signed data.
     * @throws CryptoException
     */
    byte[] sign(PrivateKey key,byte[] userId, byte[] plainText) throws CryptoException;

    /**
     * Verify the specified signature
     *
     * @param certificate the certificate of the signer as the contents of the PEM file
     * @param signatureAlgorithm the algorithm used to create the signature.
     * @param signature   the signature to verify
     * @param plainText   the original text that is to be verified
     * @return {@code true} if the signature is successfully verified; otherwise {@code false}.
     * @throws CryptoException
     */
    boolean verify(byte[] certificate, String signatureAlgorithm, byte[] signature, byte[] plainText) throws CryptoException;

    /**
     * Hash the specified text byte data.
     *
     * @param plainText the text to hash
     * @return the hashed data.
     */
    byte[] hash(byte[] plainText);
   
    /**
     * byte[]转证书
     * @param certBytes
     * @return
     * @throws CryptoException
     */
   Certificate bytesToCertificate(byte[] certBytes) throws CryptoException ;
   /**
    * 证书或公钥byte[]转公钥
    * @param certBytes
    * @return
    * @throws CryptoException
    */
   ECPublicKey bytesToPublicKey(byte[] certBytes) throws CryptoException ;
   
   /**
    * byte[]转私钥
    * @param certBytes
    * @return
    * @throws CryptoException
    */
   ECPrivateKey bytesToPrivateKey(byte[] priBytes) throws CryptoException ;
   
   /**
    * 构建证书签名请求
    * @param subject
    * @param pair
    * @return
    * @throws OperatorCreationException
    */
   PKCS10CertificationRequest generateCertificationRequest(String subject, KeyPair pair) throws OperatorCreationException;
   
   /**
    * 证书签名请求转PEM
    * @param csr
    * @return
    * @throws IOException
    */
   String certificationRequestToPEM(PKCS10CertificationRequest csr) throws IOException;
   

   //byte[] encrypto(RBCEncrypto rbcEncrypto,String colName,byte[] msg)throws CryptoException ;
   
   //byte[] decrypto(RBCEncryptoEntity encryptoEntity,String colName,byte[] msg)throws CryptoException ;


    /**
     * The CryptoSuite factory. Currently {@link #getCryptoSuite} will always
     * give you a {@link CryptoPrimitives} object
     */
    class Factory {
        private Factory() {

        }

        public static CryptoSuite getCryptoSuite() {
        	CryptoSuite cryptoSuite = null;
        	if ("sm2".equals(ChainCodeUtils.getSecurityType())){
        		cryptoSuite =  new CryptoPrimitivesSm2();
        	}else{
        		cryptoSuite =  new CryptoPrimitives();
        	}
        	try {
				cryptoSuite.init();
			} catch (Exception e) {
				e.printStackTrace();
			} 
        	return cryptoSuite;
        }
    }
}
