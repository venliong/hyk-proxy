package com.hyk.proxy.gae.server;

import java.io.IOException;
import java.io.NotSerializableException;
import java.lang.reflect.Field;
import java.net.MalformedURLException;
import java.net.URL;
import java.util.Arrays;

import javax.servlet.http.*;

import com.google.appengine.api.urlfetch.HTTPRequest;
import com.google.appengine.api.urlfetch.HTTPResponse;
import com.google.appengine.api.urlfetch.URLFetchServiceFactory;
import com.google.appengine.api.users.UserServiceFactory;
import com.hyk.compress.Compressor;
import com.hyk.compress.gz.GZipCompressor;
import com.hyk.compress.sevenzip.SevenZipCompressor;
import com.hyk.proxy.gae.common.HttpRequestExchange;
import com.hyk.proxy.gae.common.HttpResponseExchange;
import com.hyk.serializer.HykSerializer;
import com.hyk.serializer.Serializer;
import com.hyk.serializer.StandardSerializer;


@SuppressWarnings("serial")
public class FetchProxyServlet extends HttpServlet
{
	public void doPost(HttpServletRequest req, HttpServletResponse resp) throws IOException
	{
		int len = req.getContentLength();
		byte[] buffer = new byte[len];
		req.getInputStream().read(buffer);
		try
		{
			byte[] rawRes = FatchServiceWrapper.fetch(buffer);
			resp.getOutputStream().write(rawRes);
		}
		catch(Throwable e)
		{
			//e.printStackTrace();
			resp.setStatus(503);
			resp.setContentType("text/plain");
			resp.getWriter().println("####Failed " + Arrays.toString(e.getStackTrace()));
		}
		
	}
	
	public static void main(String[] args) throws NotSerializableException, IOException
	{
		HTTPRequest req = new HTTPRequest(new URL("http://www.google.com"));
		Field[] fs = req.getClass().getDeclaredFields();
		Serializer serializer = new HykSerializer();
		System.out.println("####" + serializer.serialize(req).length);
		for(Field f:fs)
		{
			System.out.println("####" + f);
		}
	}
}
