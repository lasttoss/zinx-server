using System;
using System.Collections;
using System.Net.Sockets;
using System.Text;
using UnityEngine;

public class Test : MonoBehaviour
{
    public string serverIP = "127.0.0.1";
    public int serverPort = 8999;

    private TcpClient client;
    private NetworkStream stream;
    private bool isConnected = false;

    // Start is called once before the first execution of Update after the MonoBehaviour is created
    void Start()
    {
        ConnectToServer();
        StartCoroutine(ListenForData());
    }

    void Update()
    {
        if (Input.GetKeyDown(KeyCode.Space))
        {
            Debug.Log("Sending data");
            var request = new MessageRequest(1, "ping");
            var data = MessageUtils.Encode(request);
            stream.Write(data, 0, data.Length);
        }
    }

    // Update is called once per frame
    IEnumerator ListenForData()
    {
        byte[] buffer = new byte[1024];
        while (isConnected)
        {
            if (stream.DataAvailable)
            {
                int bytesRead = stream.Read(buffer, 0, buffer.Length);
                if (bytesRead > 0)
                {
                    if (bytesRead >= 8)
                    {
                        byte[] messageIdBytes = new byte[4];
                        Array.Copy(buffer, 0, messageIdBytes, 0, 4);
                        if (BitConverter.IsLittleEndian)
                            Array.Reverse(messageIdBytes);
                        uint messageId = BitConverter.ToUInt32(messageIdBytes, 0);

                        // Read Data length (4 bytes, big-endian)
                        byte[] dataLengthBytes = new byte[4];
                        Array.Copy(buffer, 4, dataLengthBytes, 0, 4);
                        if (BitConverter.IsLittleEndian)
                            Array.Reverse(dataLengthBytes);
                        uint dataLen = BitConverter.ToUInt32(dataLengthBytes, 0);

                        if (buffer.Length >= 8 + dataLen)
                        {
                            byte[] data = new byte[dataLen];
                            Array.Copy(buffer, 8, data, 0, dataLen);
                            string message = Encoding.ASCII.GetString(data);
                            Debug.Log($"Received: {messageId} - {message}");
                        }
                    }
                }
            }

            yield return null;
        }
    }

    void ConnectToServer()
    {
        try
        {
            client = new TcpClient(serverIP, serverPort);
            stream = client.GetStream();
            isConnected = true;
            Debug.Log("Connected to server!");
        }
        catch (Exception e)
        {
            Debug.LogError("Socket error: " + e.Message);
        }
    }
}

public class MessageRequest
{
    public uint MessageId { get; set; }
    public byte[] Data { get; set; }

    public MessageRequest()
    {
    }

    public MessageRequest(uint messageId, String message)
    {
        var data = Encoding.UTF8.GetBytes(message);
        MessageId = messageId;
        Data = data;
    }
}

public static class MessageUtils
{
    public static byte[] Encode(MessageRequest message)
    {
        if (message.Data == null)
            throw new ArgumentNullException(nameof(message.Data), "Data cannot be null.");

        byte[] buffer = new byte[8 + message.Data.Length];

        // Write MessageId (4 bytes, big-endian)
        byte[] messageIdBytes = BitConverter.GetBytes(message.MessageId);
        if (BitConverter.IsLittleEndian)
            Array.Reverse(messageIdBytes);
        Array.Copy(messageIdBytes, 0, buffer, 0, 4);

        // Write Data length (4 bytes, big-endian)
        byte[] dataLengthBytes = BitConverter.GetBytes((uint)message.Data.Length);
        if (BitConverter.IsLittleEndian)
            Array.Reverse(dataLengthBytes);
        Array.Copy(dataLengthBytes, 0, buffer, 4, 4);

        // Write Data
        Array.Copy(message.Data, 0, buffer, 8, message.Data.Length);

        return buffer;
    }
}