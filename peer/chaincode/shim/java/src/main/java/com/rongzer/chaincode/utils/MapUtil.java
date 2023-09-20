package com.rongzer.chaincode.utils;

import java.util.Iterator;
import java.util.Map;

public class MapUtil {

	/**
	 * 将Source中的值合并到mapTarget中，如果target中有值，不会覆盖
	 * @param mapTarget
	 * @param mapSource
	 */
	public static void mergeMap(Map mapTarget,Map mapSource)
	{
		try
		{
			Iterator keys = mapSource.keySet().iterator();
			while (keys.hasNext()) {
				String key = (String) keys.next();
				Object value = mapSource.get(key);
				if(mapTarget.get(key) == null ||StringUtil.isEmpty(mapTarget.get(key).toString()))
				{
					try
					{
						mapTarget.put(key, value);
					}catch(Exception e1)
					{
						
					}
				}
			}
		}catch(Exception e)
		{

		}
	}
	
}
