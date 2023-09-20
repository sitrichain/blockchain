package com.rongzer.chaincode.base;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Properties;
import java.util.jar.Attributes;
import java.util.jar.JarEntry;
import java.util.jar.JarInputStream;
import java.util.jar.Manifest;

import net.sf.json.JSONObject;

import org.apache.log4j.PropertyConfigurator;
import org.bouncycastle.asn1.ASN1InputStream;
import org.bouncycastle.asn1.ASN1Sequence;
import org.bouncycastle.asn1.x509.SubjectPublicKeyInfo;
import org.bouncycastle.asn1.x509.X509CertificateStructure;
import org.apache.xerces.impl.dv.util.Base64;
import com.rongzer.blockchain.protos.peer.Rbcother.ChaincodeData;
import com.rongzer.blockchain.shim.ChaincodeBase;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.rongzer.chaincode.utils.ChainCodeUtils;

public class ChaincodeDoor extends ChaincodeBase {

	private ChaincodeBase theChainCode = null;
	private String[] args = null;
	
	@Override
	public Response init(ChaincodeStub stub) {
		logger.info(String.format("chaincode %s init by ChaincodeDoor", getId()));
		return null;
	}
	
	@Override
	public Response invoke(ChaincodeStub stub) {
		logger.debug(String.format("invoke chaincode by args %s",stub.getStringArgs().toString()));

		loadChainCode(stub);
		if (theChainCode == null){
			return newErrorResponse("error");
		}
		return theChainCode.invoke(stub);
	}
	

	
	public static void main(String[] args) throws Exception {
		if (args != null && args.length == 1 && args[0].equals("javaenv")){
			new Thread(() -> {
				logger.info("start javaenv to update");
				while (true) {
					try {
						Thread.sleep(2000);
					} catch (Exception e) {
					}
				}
			}).start();
		}else{
			ChaincodeDoor chaincodeDoor = new ChaincodeDoor();
			chaincodeDoor.args = args;
			chaincodeDoor.start(args);
		}
	}
	
	private boolean isJarLoad = false;
	private void loadChainCode(ChaincodeStub stub){
		if (theChainCode != null || isJarLoad){
			return;
		}
		isJarLoad = true;
		logger.info("start load chaincode jar");
		try{
			
			//判断加密类型
			String strCert = stub.getSecurityContext().getCallerCert().toStringUtf8();

			setSecurityType(strCert);
			//读取jar文件实便化实体
			List<String> lisArgs = new ArrayList<String>();
			lisArgs.add("getvccdata");
			lisArgs.add(stub.getSecurityContext().getChannelId());
			String chainCodeId = getId();
			String ver = "";
			if (chainCodeId.indexOf(":")>0){
				String[] chainCodeIds = chainCodeId.split(":");
				chainCodeId = chainCodeIds[0];
				ver = chainCodeIds[1];
			}
			lisArgs.add(chainCodeId);
			lisArgs.add(ver);

			Response res = stub.invokeChaincodeWithStringArgs("lscc",lisArgs);
			if (res == null || res.getPayload() == null){
				logger.error("read chaincode jar from lscc return empty");

				return ;
			}
			
			ChaincodeData cd = ChaincodeData.parseFrom(res.getPayload());
			byte[] jarBuf = cd.getCbuf().toByteArray();
			if (jarBuf ==null || jarBuf.length<100){
				logger.error(String.format("load chaincode jar buf is error"));

				return;
			}
			logger.info(String.format("load chaincode jar buf size %d",jarBuf.length));

			//加密的jar
			if (ChainCodeUtils.isCrypto(jarBuf)){
				JSONObject jGram = ChainCodeUtils.getCryptogram(stub, chainCodeId, "__CHAINCODE", "", cd.getCustomerNo(), "","");
				if (jGram != null && jGram.get(chainCodeId)!= null){
					byte[] bCryptogram = Base64.decode(jGram.getString(chainCodeId));
					jarBuf = ChainCodeUtils.dec(bCryptogram, jarBuf);
				}
			}
			
            ByteArrayInputStream is = new ByteArrayInputStream(jarBuf);
            JarInputStream jis = new JarInputStream(is);
            JarEntry entry;  
            byte[] log4jBuf = null;
            List<String> lisClassName = new ArrayList<String>();
            List<ClassLoader> lisClassLoad = new ArrayList<ClassLoader>();
            while ((entry = jis.getNextJarEntry()) != null) {  
                if (entry.getName().toLowerCase().endsWith(".class")) {  
                    String classname = entry.getName().substring(0,entry.getName().length() - ".class".length()).replace('/', '.');  
                    byte[]  classBuf = getResourceData(jis); 
                    lisClassName.add(classname);
                    ClassLoader loader = loadClass(classname,classBuf);
                    lisClassLoad.add(loader);
                }else if (entry.getName().toLowerCase().endsWith("log4j.properties")) {  
                	log4jBuf = getResourceData(jis);
                }
            }  
            for (int i=0;i<lisClassName.size();i++){
            	ClassLoader loader = lisClassLoad.get(i);
            	String className = lisClassName.get(i);
            	try{
            		Class<?> myclass = loader.loadClass(className);
            	}catch(Exception e){
            		logger.error(String.format("load class %s", e));
            	}
            }
            
            Manifest mainmanifest = jis.getManifest();
            jis.close();
            is.close();
            String mainClass = mainmanifest.getMainAttributes().getValue(Attributes.Name.MAIN_CLASS);
            Class<?> myClass = MAP_CLASS.get(mainClass);
            theChainCode = (ChaincodeBase) myClass.newInstance();  
            
            if (log4jBuf != null){
            	try{
	            	Properties props = new Properties();
	            	InputStream bis = new ByteArrayInputStream(log4jBuf);
	            	props.load(bis);
	            	PropertyConfigurator.configure(props);
	            	bis.close();
	            	logger.info("load chaincode log4j.properties success");
            	}catch(Exception e)
            	{
                    logger.error(e);
            	}
            }
            logger.info("success load chaincode jar");
		}catch(Exception e){
			logger.error(e);
		}
		
		theChainCode.processEnvironmentOptions();
		theChainCode.processCommandLineOptions(args);
	}
	

	private final static Map<String,Class<?>> MAP_CLASS = new HashMap<String,Class<?>>();
	private final static Map<String,byte[]> MAP_CLASSBUF = new HashMap<String,byte[]>();

	private static ClassLoader loadClass(String className ,byte[] classBuf){
		ClassLoader theClassLoader = Thread.currentThread().getContextClassLoader();
		MAP_CLASSBUF.put(className, classBuf);
        ClassLoader loader = new ClassLoader(theClassLoader) { 
        	
            @Override  
            public Class<?> findClass(String name) {
            	if (MAP_CLASS.get(name) != null){
            		return MAP_CLASS.get(name);
            	}
            	byte[] data = MAP_CLASSBUF.get(name);
                Class<?> clz = defineClass(name,data, 0, data.length);  
                if (clz != null){
                	MAP_CLASS.put(name, clz);
                }else{
        			logger.error(String.format("chaincode's class %s can't find", name));
                }
                return clz;  
            }  
        };

        return loader;
	}
	
    final static private byte[] getResourceData(JarInputStream jar) throws IOException {  
        ByteArrayOutputStream data = new ByteArrayOutputStream();  
        byte[] buffer = new byte[8192];  
        int size;  
        while (jar.available() > 0) {  
            size = jar.read(buffer);  
            if (size > 0) {  
                data.write(buffer, 0, size);  
            }  
        }  
        byte[] val = data.toByteArray();  
        data.close();  
        return val;  
    }
	
	/**
	 * 从证书获取公钥串
	 * @param csCert
	 * @return
	 */
	private String setSecurityType(String strCert)
    {	 
	 strCert = strCert.replaceAll("-----BEGIN -----", "");
	 strCert = strCert.replaceAll("-----END -----", "");

	 strCert = strCert.replaceAll("-----BEGIN CERTIFICATE-----", "");
	 strCert = strCert.replaceAll("-----END CERTIFICATE-----", "");
	 strCert = strCert.replaceAll("\r", "");
	 strCert = strCert.replaceAll("\n", "");

	 ASN1Sequence seq = null;
		ASN1InputStream aIn;
		try {
			ChainCodeUtils.securityType = "sm2";

			InputStream inStream = new ByteArrayInputStream(Base64.decode(strCert));

			aIn = new ASN1InputStream(inStream);
			seq = (ASN1Sequence) aIn.readObject();
			aIn.close();
			X509CertificateStructure cert = new X509CertificateStructure(seq);
			SubjectPublicKeyInfo subjectPublicKeyInfo = cert.getSubjectPublicKeyInfo();
			String params = subjectPublicKeyInfo.getAlgorithm().getParameters().toString();
			if ("1.2.840.10045.3.1.7".equals(params)){
				ChainCodeUtils.securityType = "ecdh";
			}
        }
        catch (Exception e)
        {
            e.printStackTrace();
        }
        return null;
    }


}
