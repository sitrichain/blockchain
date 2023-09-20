package com.rongzer.chaincode.utils;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.UnsupportedEncodingException;
import java.math.BigDecimal;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.text.DecimalFormat;
import java.text.NumberFormat;
import java.text.ParseException;
import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.LinkedList;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import javax.crypto.Cipher;
import javax.crypto.spec.IvParameterSpec;
import javax.crypto.spec.SecretKeySpec;

import org.apache.commons.lang.StringUtils;
import org.apache.log4j.Logger;

/* 字符串工具类
 * @page 无
 * @module COMMON
 * @author davild
 * @date 2013-04-30
 * @version 1.0
 */
public class StringUtil {
	private static Logger log = Logger.getLogger(StringUtil.class);

	private static String DATE_FORMAT = "yyyy-MM-dd";

	private static String TIME_FORMAT = "yyyy-MM-dd HH:mm:ss";

	private static int seq = 0;

	private static final int ROTATION = 99999;

	// ===========================================================================
	/**
	 * 获取包名，不包括子模块名，只到应用名，如com.yum.boh.bmp TODO 业务相关，移动到fund模块中
	 * 
	 * @param packageName
	 *            全包名
	 * @return 到应用的包名
	 */
	/*
	 * public static String getPrePackageOfPackageName(String packageName) {
	 * return FinalDefine.PRE_PACKAGE_NAME +
	 * getModelNameFormPackageName(packageName); }
	 */
	/**
	 * 获取应用名，如bmp TODO 业务相关，移动到fund模块中
	 * 
	 * @param packageName
	 *            全包名
	 * @return 应用名
	 */
	public static String getModelNameFormPackageName(String packageName) {
		packageName = packageName.substring(12);// core.util.xxx
		packageName = packageName.substring(0, packageName.indexOf("."));// core

		return packageName;
	}

	// ===========================================================================

	/**
	 * 判断字符串是否为空
	 * 
	 * @param strVal
	 *            string
	 * @return true 为空 false 不为空
	 */
	public static boolean isEmpty(String strVal) {
		return strVal == null || safeTrim(strVal).isEmpty();
	}

	/**
	 * 判断字符串是否为空
	 * 
	 * @param strVal
	 *            string
	 * @return true 不为空 false 为空
	 */
	public static boolean isNotEmpty(String strVal) {
		return !isEmpty(strVal);
	}

	/**
	 * <p>
	 * 安全的toString方法。
	 * </p>
	 * 
	 * 用于在有null风险的情况下获得对象的字符串表达。
	 * 
	 * @param obj
	 *            对象
	 * @return 如果参数为null，返回空串，否则返回参数自身的toString()方法结果
	 */
	public static String safeToString(Object obj) {
		return obj == null ? "" : obj.toString();
	}

	/**
	 * <p>
	 * 字符串加密函数。
	 * </p>
	 * 
	 * @param sSrc
	 *            加密前字符串
	 * @param sKey
	 *            密钥
	 * @return 加密后字符串
	 * @throws Exception
	 */
	public static String encrypt(String sSrc, String sKey) throws Exception {
		// 判断Key是否为空或者16位
		if (sKey == null || sKey.length() != 16) {
			return null;
		}

		byte[] raw = sKey.getBytes();
		SecretKeySpec skeySpec = new SecretKeySpec(raw, "AES");
		Cipher cipher = Cipher.getInstance("AES/CBC/PKCS5Padding");
		IvParameterSpec iv = new IvParameterSpec("0102030405060708".getBytes());
		cipher.init(Cipher.ENCRYPT_MODE, skeySpec, iv);
		byte[] encrypted = cipher.doFinal(sSrc.getBytes());

		return byte2hex(encrypted).toLowerCase();
	}

	/**
	 * <p>
	 * 字符串解密函数。
	 * </p>
	 * 
	 * @param sSrc
	 *            解密前字符串
	 * @param sKey
	 *            密钥
	 * @return 解密后字符串
	 * @throws Exception
	 */
	public static String decrypt(String sSrc, String sKey) throws Exception {
		// 判断Key是否为空或者16位
		if (sKey == null || sKey.length() != 16) {
			return null;
		}

		byte[] raw = sKey.getBytes("ASCII");
		SecretKeySpec skeySpec = new SecretKeySpec(raw, "AES");
		Cipher cipher = Cipher.getInstance("AES/CBC/PKCS5Padding");
		IvParameterSpec iv = new IvParameterSpec("0102030405060708".getBytes());
		cipher.init(Cipher.DECRYPT_MODE, skeySpec, iv);
		byte[] encrypted1 = hex2byte(sSrc);
		try {
			byte[] original = cipher.doFinal(encrypted1);
			String originalString = new String(original);
			return originalString;
		} catch (Exception e) {
			return null;
		}
	}

	/**
	 * 
	 * <p>
	 * 十六进制转byte数组
	 * </p>
	 * 
	 * @param strhex
	 *            十六进制
	 * @return byte数组
	 */
	private static byte[] hex2byte(String strhex) {
		if (strhex == null) {
			return null;
		}

		int l = strhex.length();
		if (l % 2 == 1) {
			return null;
		}
		byte[] b = new byte[l / 2];
		for (int i = 0; i != l / 2; i++) {
			b[i] = (byte) Integer.parseInt(strhex.substring(i * 2, i * 2 + 2),
					16);
		}

		return b;
	}

	/**
	 * 
	 * <p>
	 * byte数组转十六进制
	 * </p>
	 * 
	 * @param strhex
	 *            byte数组
	 * @return 十六进制
	 */
	private static String byte2hex(byte[] b) {

		String hs = "";
		String stmp = "";
		for (int n = 0; n < b.length; n++) {
			stmp = java.lang.Integer.toHexString(b[n] & 0XFF);
			if (stmp.length() == 1) {
				hs = hs + "0" + stmp;
			} else {
				hs = hs + stmp;
			}
		}

		return hs.toUpperCase();
	}

	/**
	 * 如果传入的值不为空的情况 合并字符串
	 * 
	 * @param strUrl
	 *            请求的URL地址
	 * @return String String
	 */
	public static String getPageNameByUrl(String strUrl) {
		String strResult = null;
		if (!isEmpty(strUrl)) {
			int beginIndex = strUrl.lastIndexOf("/");
			int endIndex = strUrl.lastIndexOf("_");
			int endPointIndex = strUrl.lastIndexOf(".");
			String type = null;
			if (endPointIndex != -1) {
				type = strUrl.substring(endPointIndex + 1);
			}
			if (!isEmpty(type) && type.equalsIgnoreCase("jsp")) {
				if (beginIndex > 0 && endPointIndex > 0) {
					strResult = strUrl.substring(beginIndex + 1, endPointIndex);
				}
			} else if (beginIndex > 0 && endIndex > 0) {
				strResult = strUrl.substring(beginIndex + 1, endIndex);
			}
		}

		return strResult;
	}

	/**
	 * <p>
	 * 安全的trim方法。
	 * </p>
	 * 
	 * 用于在有null风险的情况下获得{@link String#trim()}相同的效果，但不会抛出异常。<br>
	 * 如果对象不为空则将对象转换成字符串并去除空格返回,否则返回空串
	 * 
	 * @param obj
	 *            对象
	 * @return 返回非空字符串
	 */
	public static String safeTrim(Object obj) {
		char[] charArr = safeToString(obj).toCharArray();
		for (int i = 0; i < charArr.length; i++) {
			if (charArr[i] > ' ' && charArr[i] != '　') {
				break;
			} else {
				if (charArr[i] == '　') {
					charArr[i] = ' ';
				}
			}
		}
		for (int i = charArr.length - 1; i >= 0; i--) {
			if (charArr[i] > ' ' && charArr[i] != '　') {
				break;
			} else {
				if (charArr[i] == '　') {
					charArr[i] = ' ';
				}
			}
		}

		return new String(charArr).trim();
	}

	/**
	 * <p>
	 * 安全的做字符串替换
	 * </p>
	 * 
	 * 将<b>源字符串</b>中的<b>目标字符串</b>全部替换成<b>替换字符串</b> 规则如下：
	 * <ol>
	 * <li>若source为null,则结果亦 为null</li>
	 * <li>若target为null,则结果为source</li>
	 * <li>若replacement为null,则结果为source中的target全部被剔除后的新字符串</li>
	 * </ol>
	 * 
	 * @param source
	 *            源字符串
	 * @param target
	 *            目标字符串
	 * @param replacement
	 *            替换字符串
	 * @return 替换过的字符串
	 */
	public static String safeReplace(String source, String target,
			String replacement) {
		if (source == null || source.isEmpty() || target == null
				|| target.isEmpty() || target.equals(replacement)) {
			return source;
		}

		List<Integer> offsets = new ArrayList<Integer>();
		int targetLen = target.length();
		int offset = 0;
		while (true) {
			offset = source.indexOf(target, offset);
			if (offset == -1) {
				break;
			}

			offsets.add(offset);
			offset += targetLen;
		}

		String result = source;
		if (!offsets.isEmpty()) {
			// 计算结果字符串数组长度
			int sourceLen = source.length();
			if (replacement == null) {
				replacement = "";
			}

			int replacementLen = replacement.length();

			int offsetsSize = offsets.size();
			int resultLen = sourceLen + (replacementLen - targetLen)
					* offsetsSize;

			// 源/目标字符数组
			char[] sourceCharArr = source.toCharArray();
			char[] replacementCharArr = replacement.toCharArray();
			char[] destCharArr = new char[resultLen];

			// 做第一轮替换
			int firstOffset = offsets.get(0);
			System.arraycopy(sourceCharArr, 0, destCharArr, 0, firstOffset);
			if (replacementLen > 0) {
				System.arraycopy(replacementCharArr, 0, destCharArr,
						firstOffset, replacementCharArr.length);
			}

			// 中间替换
			int preOffset = firstOffset; // 前一个偏移量
			int destPos = firstOffset + replacementCharArr.length; // 目标char数组目前的有效长度(即已经填入的字符数量)
			for (int i = 1; i < offsetsSize; i++) {
				offset = offsets.get(i); // 当前偏移量
				int fragmentLen = offset - preOffset - targetLen;
				System.arraycopy(sourceCharArr, preOffset + targetLen,
						destCharArr, destPos, fragmentLen);
				destPos += fragmentLen;
				if (replacementLen > 0) {
					System.arraycopy(replacementCharArr, 0, destCharArr,
							destPos, replacementCharArr.length);
				}
				preOffset = offset;
				destPos += replacementCharArr.length;
			}

			// 做末轮替换
			int lastFragmentLen = sourceLen - preOffset - targetLen;
			System.arraycopy(sourceCharArr, preOffset + targetLen, destCharArr,
					destPos, lastFragmentLen);

			result = new String(destCharArr);
		}

		return result;
	}

	/**
	 * 检查字符串中是否包含某个字符
	 * 
	 * @param str
	 *            需要检查的字符
	 * @param compare
	 *            字符串
	 * @return true 包含改字符 false 不包含改字符
	 */
	public static boolean contains(String str, String compare) {
		boolean bool = true;
		if (isEmpty(str)) {
			bool = false;
		} else {
			if (str.indexOf(compare) < 0) {
				bool = false;
			}
		}

		return bool;
	}

	/**
	 * 检查字符串中是否包含某个字符
	 * 
	 * @param str
	 *            需要检查的字符
	 * @param compare
	 *            字符串
	 * @return true 不包含改字符 false 包含改字符
	 */
	public static boolean notContains(String str, String compare) {
		return !contains(str, compare);
	}

	/**
	 * 将首字符大写
	 * 
	 * @param str
	 *            字符串
	 * @return 首字符大写的字符串
	 */
	public static String toFirstLetterUpperCase(String str) {
		if (StringUtil.isNotEmpty(str)) {
			String firstLetter = str.substring(0, 1).toUpperCase();
			return firstLetter + str.substring(1, str.length());
		} else {
			return str;
		}
	}

	/**
	 * <PRE>
	 * 转换字符串为Short
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return 转换后的Short，失败是返回0
	 */
	public static Short toShort(Object value) throws NumberFormatException {
		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return Short.valueOf(safeTrim(val));
			} else {
				return null;
			}
		} catch (NumberFormatException e) {
			throw e;
		}
	}

	/**
	 * 字符串取整
	 * 
	 * @param value
	 * @return
	 */
	public static int toInt(String value) {
		return toInt(value, -1);
	}

	/**
	 * 字符串取整
	 * 
	 * @param value
	 * @return
	 */
	public static String calIntPerValue(Object value, int perValue, int allValue) {
		if (value == null) {
			value = "";
		}
		String strReturn = value.toString();
		try {
			float f = toFloat(value).floatValue();
			f = f * perValue / allValue;
			BigDecimal b = BigDecimal.valueOf(f);
			float f1 = b.setScale(0, BigDecimal.ROUND_HALF_UP).floatValue();

			int nValue = 1 * (int) f1;
			strReturn = "" + nValue;
		} catch (Exception e) {

		}

		return strReturn;
	}

	/**
	 * 字符串取整
	 * 
	 * @param value
	 * @return
	 */
	public static String calFixPerValue(Object value, int perValue, int allValue) {
		String strReturn = value.toString();
		try {
			float f = toFloat(value).floatValue();
			f = f * perValue / allValue;
			BigDecimal b = BigDecimal.valueOf(f);
			float f1 = b.setScale(2, BigDecimal.ROUND_HALF_UP).floatValue();
			strReturn = "" + f1;
		} catch (Exception e) {

		}

		return strReturn;
	}

	/**
	 * 字符串取整
	 * 
	 * @param value
	 * @return
	 */
	public static int toInt(String value, int nDefault) {
		
		
		int nReturn = nDefault;
		try {
			if (value != null) {
				String strValue = value.toString();
				//去除千分位
				strValue = strValue.replaceAll(",", "");
				if (strValue.indexOf(".") >= 0) {
					strValue = strValue.substring(0, strValue.indexOf("."));
				}

				nReturn = toInteger(strValue);

			}

		} catch (Exception e) {

		}
		return nReturn;
	}

	/**
	 * <PRE>
	 * 转换字符串为Integer
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return 转换后的Integer，失败是返回0
	 */
	public static Integer toInteger(Object value) throws NumberFormatException {

		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return Integer.valueOf(safeTrim(val));
			} else {
				return null;
			}
		} catch (NumberFormatException e) {
			throw e;
		}
	}

	/**
	 * <PRE>
	 * 
	 * 如果输入null则转变为空,其它情况直接toString()
	 * 
	 * </PRE>
	 * 
	 * @param inObject
	 *            入力对象
	 * @return String 字符
	 */
	public static String toStringWithEmpty(Object inObject) {
		if (inObject == null) {
			return "";
		} else {
			return inObject.toString();
		}
	}

	/**
	 * <PRE>
	 * 转换字符串为Integer
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return 转换后的Integer，失败是返回0
	 */
	public static Integer toIntegerwithZero(Object value)
			throws NumberFormatException {
		String val = toStringWithEmpty(value);
		try {
			if (isEmpty(val)) {
				return 0;
			} else {
				return Integer.valueOf(safeTrim(val));
			}
		} catch (NumberFormatException e) {
			throw e;
		}
	}

	/**
	 * <PRE>
	 * 转换字符串为Long
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @param defaultlong
	 *            long
	 * @return 转换后的Long，失败是返回-1
	 */
	public static long tolong(Object value, long defaultlong) {
		long lValue = -1;
		try {
			lValue = toLong(value).longValue();
		} catch (Exception e) {

		}

		return lValue;

	}

	/**
	 * <PRE>
	 * 转换字符串为Long
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return 转换后的Long，失败是返回0
	 */
	public static Long toLong(Object value) throws NumberFormatException {
		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return Long.valueOf(safeTrim(val));
			} else {
				return null;
			}
		} catch (NumberFormatException e) {
			throw e;
		}
	}

	/**
	 * 字符转浮点数
	 * 
	 * @param value
	 * @param defaultValue
	 * @return
	 */
	public static float toFloat(Object value, float defaultValue) {
		float fValue = defaultValue;
		try {
			fValue = toFloat(value).floatValue();
		} catch (Exception e) {

		}

		return fValue;
	}

	/**
	 * <PRE>
	 * 转换字符串为Float
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return 转换后的Double，失败是返回0
	 */
	public static Float toFloat(Object value) throws NumberFormatException {
		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return Float.valueOf(safeTrim(val));
			} else {
				return null;
			}
		} catch (NumberFormatException e) {
			throw e;
		}
	}

	/**
	 * 字符转浮点数
	 * 
	 * @param value
	 * @param defaultValue
	 * @return
	 */
	public static double toDouble(String value, double defaultValue) {
		double dValue = defaultValue;
		try {
			dValue = toDouble(value).floatValue();
		} catch (Exception e) {

		}

		return dValue;
	}

	/**
	 * <PRE>
	 * 转换字符串为Double
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return 转换后的Double，失败是返回0
	 */
	public static Double toDouble(Object value) throws Exception {
		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return Double.valueOf(safeTrim(val));
			} else {
				return null;
			}
		} catch (NumberFormatException e) {
			throw e;
		}
	}

	/**
	 * <PRE>
	 * 转换字符串为BigDecimal
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return BigDecimal 转换后的BigDecimal，失败是返回0
	 */
	public static BigDecimal toBigDecimal(Object value) throws Exception {
		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return new BigDecimal(safeTrim(val));
			} else {
				return null;
			}

		} catch (Exception e) {
			throw e;
		}
	}

	/**
	 * <PRE>
	 * 转换字符串为BigDecimal
	 * </PRE>
	 * 
	 * @param value
	 *            String
	 * @throws StringToNumberException
	 *             异常
	 * @return BigDecimal 转换后的BigDecimal，失败是返回0
	 */
	public static BigDecimal toBigDecimal1(Object value) throws Exception {
		String val = value != null ? String.valueOf(value) : null;
		try {
			if (isNotEmpty(val)) {
				return new BigDecimal(safeTrim(val));
			} else {
				return null;
			}
		} catch (Exception e) {
			throw e;
		}
	}

	/**
	 * <PRE>
	 * 
	 * 4舍5入,返回不含科学计数法的字符
	 * 
	 * </PRE>
	 * 
	 * @param value
	 *            值
	 * @param nScal
	 *            精度
	 * @return String 4舍5入后的值
	 */
	public static String toDoubleNoE(double value, int nScal) {
		DecimalFormat numberForm = (DecimalFormat) NumberFormat
				.getNumberInstance(Locale.CHINA);
		numberForm.setGroupingUsed(false);
		numberForm.setMaximumFractionDigits(nScal);
		String strReturn = "";
		if (nScal >0)
		{
			String tmp = "%."+nScal+"f";
			strReturn = String.format(tmp, value);
		}else
		{
			strReturn = numberForm.format(value);
		}
		return strReturn;
	}

	/**
	 * 将类名中的下划线去掉，适用于在Entities对象中的key的类名重复的时候加上下划线的情况
	 * 
	 * @param value
	 *            带下划线的ClassName或者不带的
	 * @return 去掉下划线的ClassName，比如SampleDAO_1，返回则为SampleDAO
	 */
	/*
	 * public static String filterUnderLineForClassName(String value) { String
	 * returnValue = null; if (StringUtil.isNotEmpty(value) &&
	 * value.indexOf(FinalDefine.SPLIT_STRING_UNDERLINE) >= 0) { returnValue =
	 * value.substring(0, value.indexOf(FinalDefine.SPLIT_STRING_UNDERLINE)); }
	 * else { returnValue = value; } return returnValue; }
	 */

	/**
	 * <PRE>
	 * 转换字符串为Boolean
	 * </PRE>
	 * 
	 * @param value
	 *            0,1)
	 * @return Boolean 1-True/非1-False
	 */
	public static Boolean toBoolean(String value) {
		if ("1".equals(value)) {
			return Boolean.TRUE;
		} else {
			return Boolean.FALSE;
		}
	}

	/**
	 * 功能：验证字符串长度是否符合要求，一个汉字等于两个字符
	 * 
	 * @param strParameter
	 *            要验证的字符串
	 * @return 符合长度ture 超出范围false
	 */
	public static int validateStrByLength(String strParameter) {
		int temp_int = 0;
		byte[] b = strParameter.getBytes();

		for (int i = 0; i < b.length; i++) {
			if (b[i] >= 0) {
				temp_int = temp_int + 1;
			} else {
				temp_int = temp_int + 2;
				i++;
			}
		}
		return temp_int;
	}

	/**
	 * 简易分隔字符串
	 * 
	 * @param source
	 * @return
	 */
	public static List<String> split(String source) {
		List<String> list = new ArrayList<String>();
		if (isNotEmpty(source)) {
			String[] strs = source.split(",");
			for (String str : strs) {
				list.add(str);

			}
		}
		return list;
	}

	/**
	 * <p>
	 * 字符串快速分割。
	 * </p>
	 * 
	 * <p>
	 * 提供更高效简洁的分割算法，在相同情况下，分割效率约为<br>
	 * {@link java.lang.String#split(String)}的4倍<br>
	 * 但仅支持单个字符作为分割符。
	 * </p>
	 * 
	 * <p>
	 * 当待分割字符串为null或者空串时，返回长度为0的数组<br>
	 * 该方法不会抛出任何异常
	 * </p>
	 * 
	 * @param source
	 *            待分割字符串
	 * @param splitChar
	 *            分隔符
	 * @return 分割完毕的字符串数组
	 */
	public static String[] split(String source, char splitChar) {
		String[] strArr = null;
		List<String> strList = new LinkedList<String>();
		if (null == source || source.isEmpty()) {
			strArr = new String[0];
		} else {
			char[] charArr = source.toCharArray();
			int start = 0;
			int end = 0;
			while (end < source.length()) {
				char c = charArr[end];
				if (c == splitChar) {
					if (start != end) {
						String fragment = source.substring(start, end);
						strList.add(fragment);
					}
					start = end + 1;
				}
				++end;
			}
			if (start < source.length()) {
				strList.add(source.substring(start));
			}

			strArr = new String[strList.size()];
			strList.toArray(strArr);
		}

		return strArr;
	}

	/**
	 * <p>
	 * 字符串快速分割。
	 * </p>
	 * 
	 * <p>
	 * 提供更高效简洁的分割算法，在相同情况下，分割效率约为<br>
	 * {@link java.lang.String#split(String)}的4倍<br>
	 * 支持字符串作为分割符。
	 * </p>
	 * 
	 * <p>
	 * 当待分割字符串为null或者空串时，返回长度为0的数组<br>
	 * 该方法不会抛出任何异常<br>
	 * 如果传入的字符串中仅含有1个字符，那么该方法会被转交给{@link #split(String, char)}处理
	 * </p>
	 * 
	 * @param source
	 *            待分割字符串
	 * @param splitStr
	 *            分隔符字符串
	 * @return 分割完毕的字符串数组
	 * 
	 * @see #split(String, char)
	 */
	public static String[] split(String source, String splitStr) {
		String[] strArr = null;
		if (null == source || source.isEmpty()) {
			strArr = new String[0];
		} else {
			if (splitStr.length() == 1) {
				strArr = split(source, splitStr.charAt(0));
			} else {
				int strLen = source.length();
				int splitStrLen = splitStr.length();
				List<String> strList = new LinkedList<String>();
				int start = 0;
				int end = 0;
				while (start < strLen) {
					end = source.indexOf(splitStr, start);
					if (end == -1) {
						String fregment = source.substring(start);
						strList.add(fregment);
						break;
					}
					if (start != end) {
						String fregment = source.substring(start, end);
						strList.add(fregment);
					}
					start = end + splitStrLen;
				}

				strArr = new String[strList.size()];
				strList.toArray(strArr);
			}
		}

		return strArr;
	}

	/**
	 * 将长时间格式字符串转换为时间 yyyy-MM-dd HH:mm:ss
	 * 
	 * @param strDate
	 * @return
	 */
	public static Date strToDate(String strDate) {
		Date strtodate = null;
		try {
			SimpleDateFormat formatter = new SimpleDateFormat("yyyy-MM-dd");
			ParsePosition pos = new ParsePosition(0);
			strtodate = formatter.parse(strDate, pos);
		} catch (Exception e) {

		}

		return strtodate;
	}

	/**
	 * 将长时间格式字符串转换为时间 yyyy-MM-dd HH:mm:ss
	 * 
	 * @param strDate
	 * @return
	 */
	public static Date strToDateLong(String strDate) {
		Date strtodate = null;
		try {
			SimpleDateFormat formatter = new SimpleDateFormat(
					"yyyy-MM-dd HH:mm:ss");
			ParsePosition pos = new ParsePosition(0);
			strtodate = formatter.parse(strDate, pos);
		} catch (Exception e) {

		}

		return strtodate;
	}

	/**
	 * 将短时间格式时间转换为字符串 yyyy-MM-dd
	 * 
	 * @param dateDate
	 * @return
	 */
	public static String dateToStr(Date dateDate) {
		String dateString = "";
		try {
			SimpleDateFormat formatter = new SimpleDateFormat("yyyy-MM-dd");
			dateString = formatter.format(dateDate);
		} catch (Exception e) {

		}

		return dateString;
	}

	/**
	 * 将短时间格式时间转换为字符串 yyyy-MM-dd
	 * 
	 * @param dateDate
	 * @return
	 */
	public static String dateTimeToStr(Date dateDate) {
		String dateString = "";
		try {
			SimpleDateFormat formatter = new SimpleDateFormat(
					"yyyy-MM-dd HH:mm:ss");
			dateString = formatter.format(dateDate);
		} catch (Exception e) {

		}

		return dateString;
	}

	/**
	 * 将短时间格式时间转换为字符串 yyyy-MM-dd
	 * 
	 * @param dateDate
	 * @param format
	 * @return
	 */
	public static String dateToStr(Date dateDate, String format) {
		SimpleDateFormat formatter = new SimpleDateFormat(format);
		String dateString = formatter.format(dateDate);
		return dateString;
	}

	/**
	 * 特殊的日期转换格式针对iceData控件
	 * 
	 * @param dateDate
	 * @return 正确格式的日期
	 * @throws ParseException
	 */
	public static String dateToStringToStr(String dateDate, String format) {
		SimpleDateFormat sdf = new SimpleDateFormat(
				"EEE MMM dd HH:mm:ss zzz yyyy", Locale.US);
		Date dateString = null;
		try {
			dateString = sdf.parse(dateDate);
		} catch (ParseException e) {
			try {
				dateString = new Date(Long.parseLong(dateDate));
			} catch (NumberFormatException e1) {
				if (log.isDebugEnabled()) {
					log.debug("检测到数据转换错误,StringUtil.dateToStringToStr");
				}
				return null;
			}
		}
		if (null != dateString) {
			if (StringUtil.isNotEmpty(format)) {
				return StringUtil.dateToStr(dateString, format);
			} else {
				return StringUtil.dateToStr(dateString, "yyyy-MM-dd HH:mm");
			}
		}
		return null;
	}

	/**
	 * 
	 * <p>
	 * 取得16进制字符串。
	 * </p>
	 * 
	 * @param b
	 *            byte数组
	 * @return 16进制字符串
	 */
	public static String getHexString(byte[] b) {
		String result = "";
		for (int i = 0; i < b.length; i++) {
			result += Integer.toString((b[i] & 0xff) + 0x100, 16).substring(1);
		}
		return result;
	}

	/**
	 * 
	 * <p>
	 * 将16进制字符串转换为byte数组。
	 * </p>
	 * 
	 * @param str
	 *            16进制字符串
	 * @return byte数组
	 */
	public static byte[] getByteFromHexString(String str) {
		byte[] result = null;
		try {
			if (StringUtil.isNotEmpty(str) && str.length() > 0
					&& str.length() % 2 == 0) {
				result = new byte[str.length() / 2];

				String temp = str;
				int i = 0;
				for (i = 0; temp.length() > 2; i++) {
					result[i] = (byte) Integer.parseInt(temp.substring(0, 2),
							16);
					temp = temp.substring(2);
				}
				if (temp.length() == 2) {
					result[i] = (byte) Integer.parseInt(temp, 16);
				}
			}
		} catch (Exception ex) {
		}

		return result;
	}

	public static String subTwentyChar(String text) {
		String resultStr = "";
		int i = 0;
		while (validateStrByLength(resultStr) != 20
				&& validateStrByLength(resultStr) != 21
				&& validateStrByLength(resultStr) != 22) {
			i++;
			resultStr = text.substring(0, i);
		}
		return resultStr;
	}

	/**
	 * 32 uuid
	 * 
	 * @return
	 */
	public static String getUuid32() {
		return java.util.UUID.randomUUID().toString().replace("-", "");
	}

	/**
	 * 获取当前时间的字符串
	 * 
	 * @return
	 */
	public static String getNowDate() {
		Date objDate = new Date();
		return getDate(objDate);
	}

	/**
	 * 
	 * @param strDate
	 */
	public static Date getDate(String strDate) {
		Date date = new Date();
		if (strDate != null) {
			SimpleDateFormat dt = new SimpleDateFormat(DATE_FORMAT);
			try {
				date = dt.parse(strDate);
			} catch (ParseException e) {
			}
		}

		return date;
	}

	/**
	 * 
	 * @param strDate
	 */
	public static Date getTime(String strTime) {
		Date date = new Date();
		if (strTime != null) {
			SimpleDateFormat dt = new SimpleDateFormat(TIME_FORMAT);
			try {
				date = dt.parse(strTime);
			} catch (ParseException e) {
			}
		}

		return date;
	}

	/**
	 * 
	 * @param strDate
	 */
	public static Date getDate(Date objDate, String addDay) {
		Calendar lastDate = Calendar.getInstance();
		int nAddDate = 0;
		try {
			nAddDate = Integer.parseInt(addDay);

		} catch (Exception e) {

		}
		try {
			lastDate.setTime(objDate); // 设置当前日期
			lastDate.add(Calendar.DATE, nAddDate);
			
		} catch (Exception e) {
		}

		return lastDate.getTime();
	}

	/**
	 * 获取时间的字符串
	 * 
	 * @param objDate
	 * @return
	 */
	public static String getDate(Date objDate) {
		String strReturn = "";
		if (objDate != null) {
			SimpleDateFormat dt = new SimpleDateFormat(DATE_FORMAT);
			strReturn = dt.format(objDate);
		}

		return strReturn;
	}

	/**
	 * 获取当前时间的字符串
	 * 
	 * @return
	 */
	public static String getNowTime() {
		Date objDate = new Date();
		return getTime(objDate);
	}

	/**
	 * 获取当前时间的字符串END
	 * 
	 * @return
	 */
	public static String getNowTimeS() {
		Date objDate = new Date();
		return getDate(objDate) + " 00:00:00";
	}

	/**
	 * 获取当前时间的字符串END
	 * 
	 * @return
	 */
	public static String getNowTimeE() {
		Date objDate = new Date();
		return getDate(objDate) + " 23:59:59";
	}

	/**
	 * 获取时间的字符串
	 * 
	 * @param objDate
	 * @return
	 */
	public static String getTime(Date objDate) {
		String strReturn = "";
		if (objDate != null) {
			SimpleDateFormat dt = new SimpleDateFormat(TIME_FORMAT);
			strReturn = dt.format(objDate);
		}

		return strReturn;
	}

	
	/**
	 * 获取时间的字符串
	 * 
	 * @param objDate
	 * @return
	 */
	public static Date getTime(Date objDate ,int addSeconds) {
		
		Calendar lastDate = Calendar.getInstance();

		try {
			lastDate.setTime(objDate); // 设置当前日期
			lastDate.add(Calendar.SECOND, addSeconds);

		} catch (Exception e) {
		}

		return lastDate.getTime();
	}
	
	/**
	 * 取｛与｝中间匹配的通配符
	 * 
	 * @param strInput
	 * @return
	 */
	public static List<String> getReplaceFlag(String strInput) {
		List<String> lisReturn = new ArrayList<String>();

		return getReplaceFlag(strInput, lisReturn);
	}

	/**
	 * 取｛与｝中间匹配的通配符,只匹配有的值
	 * 
	 * @param strInput
	 * @return
	 */
	public static String replaceParamHasValue(String strInput, Map map) {
		String strReturn = strInput;
		List<String> lisParam = getReplaceFlag(strInput);
		if (lisParam != null && lisParam.size() > 0 && map != null) {
			for (String strParam : lisParam) {
				String strValue = "";
				Object objValue = map.get(strParam);
				if (objValue != null) {
					strValue = objValue.toString();
					if (StringUtil.isNotEmpty(strValue)) {
						strReturn = strReturn.replaceAll("\\{" + strParam
								+ "\\}", strValue);
					}
				}
			}
		}
		return strReturn;
	}

	/**
	 * 取｛与｝中间匹配的通配符
	 * 
	 * @param strInput
	 * @return
	 */
	public static String replaceParam(String strInput, Map map) {
		String strReturn = strInput;
		List<String> lisParam = getReplaceFlag(strInput);
		if (lisParam != null && lisParam.size() > 0 && map != null) {
			for (String strParam : lisParam) {
				String strValue = "";
				Object objValue = map.get(strParam);
				if (objValue != null) {
					strValue = objValue.toString();
				}
				strReturn = strReturn.replaceAll("\\{" + strParam + "\\}",
						strValue);
			}
		}
		return strReturn;
	}

	/**
	 * 取｛与｝中间匹配的通配符
	 * 
	 * @param strInput
	 * @return
	 */
	private static List<String> getReplaceFlag(String strInput,
			List<String> lisReturn) {
		if (strInput != null) {
			int nSIndex = strInput.indexOf("{");
			if (nSIndex >= 0) {
				int nEIndex = strInput.indexOf("}", nSIndex);
				if (nEIndex > nSIndex) {
					lisReturn.add(strInput.substring(nSIndex + 1, nEIndex));
					getReplaceFlag(strInput.substring(nEIndex + 1), lisReturn);
				}
			}
		}
		return lisReturn;
	}

	/**
	 * 格式化数字为千分位显示；
	 * 
	 * @param 要格式化的数字
	 *            ；
	 * @return
	 */
	public static String FormatMicrometer(String strInput) {
		DecimalFormat df = null;
		if (strInput.indexOf(".") > 0) {
			if (strInput.length() - strInput.indexOf(".") - 1 == 0) {
				df = new DecimalFormat("###,##0.");
			} else if (strInput.length() - strInput.indexOf(".") - 1 == 1) {
				df = new DecimalFormat("###,##0.00");
			} else {
				df = new DecimalFormat("###,##0.00");
			}
		} else {
			df = new DecimalFormat("###,##0");
		}
		double number = 0.0;
		try {
			number = Double.parseDouble(strInput);
		} catch (Exception e) {
			number = 0.0;
		}
		return df.format(number);
	}

	private static String htmlEncode(char c) {

		switch (c) {

		case '&':

			return "&amp;";

		case '<':

			return "&lt;";

		case '>':

			return "&gt;";
		case '"':

			return "&quot;";
		case '\'':

			return "&#39;";
		case '/':

			return "&#x2F;";
		default:

			return c + "";

		}

	}

	/** 对传入的字符串str进行Html encode转换 */
	public static String htmlEncode(String str) {

		if (str == null || str.trim().equals(""))
			return str;

		StringBuilder encodeStrBuilder = new StringBuilder();

		for (int i = 0, len = str.length(); i < len; i++) {

			encodeStrBuilder.append(htmlEncode(str.charAt(i)));

		}

		return encodeStrBuilder.toString();

	}

	/** 对传入的字符串str进行Html decode转换 */
	public static String htmlDecode(String str) {
		String strReturn = str;
		try {
			strReturn = safeReplace(strReturn, "&amp;", "&");

			strReturn = safeReplace(strReturn, "&gt;", ">");
			strReturn = safeReplace(strReturn, "&lt;", "<");
			strReturn = safeReplace(strReturn, "&quot;", "\"");
			strReturn = safeReplace(strReturn, "&#39;", "'");
			strReturn = safeReplace(strReturn, "&#x2F;", "/");
		} catch (Exception e) {

		}
		return strReturn;

	}

	private static final String regEx_script = "<script[^>]*?>[\\s\\S]*?<\\/script>"; // 定义script的正则表达式
	private static final String regEx_style = "<style[^>]*?>[\\s\\S]*?<\\/style>"; // 定义style的正则表达式
	private static final String regEx_html = "<[^>]+>"; // 定义HTML标签的正则表达式
	private static final String regEx_space = "\\s*|\t|\r|\n";// 定义空格回车换行符

	/**
	 * @param htmlStr
	 * @return 删除Html标签
	 */
	public static String delHTMLTag(String htmlStr) {
		if (htmlStr == null) {
			htmlStr = "";
		}
		try {
			Pattern p_script = Pattern.compile(regEx_script,
					Pattern.CASE_INSENSITIVE);
			Matcher m_script = p_script.matcher(htmlStr);
			htmlStr = m_script.replaceAll(""); // 过滤script标签
		} catch (Exception e) {

		}

		try {
			Pattern p_style = Pattern.compile(regEx_style,
					Pattern.CASE_INSENSITIVE);
			Matcher m_style = p_style.matcher(htmlStr);
			htmlStr = m_style.replaceAll(""); // 过滤style标签
		} catch (Exception e) {

		}

		try {
			Pattern p_html = Pattern.compile(regEx_html,
					Pattern.CASE_INSENSITIVE);
			Matcher m_html = p_html.matcher(htmlStr);
			htmlStr = m_html.replaceAll(""); // 过滤html标签
		} catch (Exception e) {

		}
		/*
		 * try {
		 * 
		 * //Pattern p_space = Pattern.compile(regEx_space,
		 * Pattern.CASE_INSENSITIVE); //Matcher m_space =
		 * p_space.matcher(htmlStr); //htmlStr = m_space.replaceAll(""); //
		 * 过滤空格回车标签 }catch (Exception e) {
		 * 
		 * }
		 */

		return htmlStr.trim(); // 返回文本字符串
	}

	public static String getTextFromHtml(String htmlStr) {
		htmlStr = delHTMLTag(htmlStr);
		htmlStr = htmlStr.replaceAll("&nbsp;", "");
		//htmlStr = htmlStr.substring(0, htmlStr.indexOf("。") + 1);
		return htmlStr;
	}

	   /** 
	    * md5加密(ITS) 支持中文
	    * @param str 
	    * @param charSet 字符编码
	    * @return 
	    */  
	   public synchronized static final String MD5(String str,String charSet) { //md5加密  
	    MessageDigest messageDigest = null;    
	    try {    
	        messageDigest = MessageDigest.getInstance("MD5");    
	        messageDigest.reset();   
	        if(charSet==null){  
	            messageDigest.update(str.getBytes());  
	        }else{  
	            messageDigest.update(str.getBytes(charSet));    
	        }             
	    } catch (NoSuchAlgorithmException e) {    
	        log.error("md5 error:"+e.getMessage(),e);  
	    } catch (UnsupportedEncodingException e) {    
	        log.error("md5 error:"+e.getMessage(),e);  
	    }    
	      
	    byte[] byteArray = messageDigest.digest();    
	    StringBuffer md5StrBuff = new StringBuffer();    
	    for (int i = 0; i < byteArray.length; i++) {                
	        if (Integer.toHexString(0xFF & byteArray[i]).length() == 1)    
	            md5StrBuff.append("0").append(Integer.toHexString(0xFF & byteArray[i]));    
	        else    
	            md5StrBuff.append(Integer.toHexString(0xFF & byteArray[i]));    
	    }    
	    return md5StrBuff.toString();    
	} 
   
	/***
	 * MD5加码 生成32位md5码
	 */
	public static String MD5(String inStr) {
		MessageDigest md5 = null;
		byte[] byteArray = null;
		try {
			md5 = MessageDigest.getInstance("MD5");
			byteArray = inStr.getBytes("UTF-8");

		} catch (Exception e) {
			System.out.println(e.toString());
			e.printStackTrace();
			return "";
		}

		byte[] md5Bytes = md5.digest(byteArray);
		StringBuffer hexValue = new StringBuffer();
		for (int i = 0; i < md5Bytes.length; i++) {
			int val = ((int) md5Bytes[i]) & 0xff;
			if (val < 16)
				hexValue.append("0");
			hexValue.append(Integer.toHexString(val));
		}
		return hexValue.toString();

	}
	
	private static final String hexDigits[] = { "0", "1", "2", "3", "4", "5",
			"6", "7", "8", "9", "a", "b", "c", "d", "e", "f" };	
	private static String byteToHexString(byte b) {
		int n = b;
		if (n < 0)
			n += 256;
		int d1 = n / 16;
		int d2 = n % 16;
		return hexDigits[d1] + hexDigits[d2];
	}

	private static String byteArrayToHexString(byte b[]) {
		StringBuffer resultSb = new StringBuffer();
		for (int i = 0; i < b.length; i++)
			resultSb.append(byteToHexString(b[i]));

		return resultSb.toString();
	}
	/**
	 * 微信签名MD5
	 * @param origin
	 * @param charsetname
	 * @return
	 */
	public static String MD5Encode(String origin, String charsetname) {
		String resultString = null;
		try {
			resultString = new String(origin);
			MessageDigest md = MessageDigest.getInstance("MD5");
			if (charsetname == null || "".equals(charsetname))
				resultString = byteArrayToHexString(md.digest(resultString
						.getBytes()));
			else
				resultString = byteArrayToHexString(md.digest(resultString
						.getBytes(charsetname)));
		} catch (Exception exception) {
		}
		return resultString;
	}

	
	/**
	 * 按日期生成5位验证码,100秒内有效
	 * @param inStr
	 * @return
	 */
	public static String createValidKey(String inStr)
	{
		String nowDate = ""+new Date().getTime();
		nowDate = nowDate.substring(0, nowDate.length()-4);
		String str = inStr + nowDate + inStr + "1" + inStr;
		return  MD5(str).substring(3, 8).toLowerCase();
				
	}

	/**
	 * 按秒取序号
	 * 
	 * @return
	 */
	private static synchronized String nextSeq() {
		if (seq > ROTATION)
			seq = 0;
		StringBuilder buf = new StringBuilder();

		buf.delete(0, buf.length());
		Date date = new Date();
		// 每分钟10万单
		String str = String.format("%1$ty%1$tm%1$td%1$tH%1$tM%2$05d", date,
				seq++);
		return str;
	}

	/**
	 * 
	 * @param end
	 * @return
	 */
	public static String getSeqNum(String end) {
		String strSeq = nextSeq();
		strSeq = strSeq + end;
		return strSeq;
	}

	/**
	 * 判断是否为IP
	 * 
	 * @param end
	 * @return
	 */
	public static boolean checkIP(String str) {
		Pattern pattern = Pattern
				.compile("^((\\d|[1-9]\\d|1\\d\\d|2[0-4]\\d|25[0-5]"
						+ "|[*])\\.){3}(\\d|[1-9]\\d|1\\d\\d|2[0-4]\\d|25[0-5]|[*])$");
		return pattern.matcher(str).matches();
	}

	public static String inputStreamToString(InputStream is) {

		BufferedReader reader = null;
		try {
			reader = new BufferedReader(new InputStreamReader(is,"UTF-8"));
		} catch (Exception e1) {
			e1.printStackTrace();
		}
		
		StringBuilder sb = new StringBuilder();
		String line = null;
		try {

			while ((line = reader.readLine()) != null) {
				sb.append(line + "\n");
			}

		} catch (IOException e) {
			e.printStackTrace();
		} finally {
			try {
				is.close();
			} catch (IOException e) {
				e.printStackTrace();
			}
		}
		return sb.toString();
	}
	
	  /**
     * 检测是否有emoji字符
     * @param source
     * @return 一旦含有就抛出
     */
    public static boolean containsEmoji(String source) {
        if (StringUtils.isEmpty(source)) {
            return false;
        }
        
        int len = source.length();
        
        for (int i = 0; i < len; i++) {
            char codePoint = source.charAt(i);
            
            if (isEmojiCharacter(codePoint)) {
                //do nothing，判断到了这里表明，确认有表情字符
                return true;
            }
        }
        
        return false;
    }

	
  private static boolean isEmojiCharacter(char codePoint) {
        return (codePoint == 0x0) || 
                (codePoint == 0x9) ||                            
                (codePoint == 0xA) ||
                (codePoint == 0xD) ||
                ((codePoint >= 0x20) && (codePoint <= 0xD7FF)) ||
                ((codePoint >= 0xE000) && (codePoint <= 0xFFFD)) ||
                ((codePoint >= 0x10000) && (codePoint <= 0x10FFFF));
    }
	
	/**
     * 过滤emoji 或者 其他非文字类型的字符
     * @param source
     * @return
     */
    public static String filterEmoji(String source) {
        
        if (!containsEmoji(source)) {
            return source;//如果不包含，直接返回
        }
        //到这里铁定包含
        StringBuilder buf = null;
        
        int len = source.length();
        
        for (int i = 0; i < len; i++) {
            char codePoint = source.charAt(i);
            
            if (isEmojiCharacter(codePoint)) {
                if (buf == null) {
                    buf = new StringBuilder(source.length());
                }
                
                buf.append(codePoint);
            } else {
            }
        }
        
        if (buf == null) {
            return source;//如果没有找到 emoji表情，则返回源字符串
        } else {
            if (buf.length() == len) {//这里的意义在于尽可能少的toString，因为会重新生成字符串
                buf = null;
                return source;
            } else {
                return buf.toString();
            }
        }
        
    }
    
    /**double 判断是否相等
     * @param a
     * @param b
     * @return
     */
    public static boolean doubleEquals(double a, double b)
    {
    	double d = 0.0001;
    	if(a - b > -d && a - b < d){
    		return true;
    	}else{
    		return false;
    	}
    }
    
    
    

}