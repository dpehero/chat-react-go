import React, { useState, useEffect } from 'react';
import { w3cwebsocket as W3CWebSocket } from 'websocket';

interface IChat {
  message: string
}

const client = new W3CWebSocket('ws://localhost:8080/ws');

const Chat = () => {
  const [message, setMessage] = useState<string>('');
  const [chat, setChat] = useState<IChat[]>([]);

  useEffect(() => {
    // Fetch chat messages from your server when the component mounts
    
      fetchMessages();
    client.onopen = () => {
      console.log('Connected to WebSocket');
    };
    
    client.onmessage = (message: any) => {
      // setChat([message.data,...chat]);
      // const newMessage = JSON.parse(message.data);
      const t = chat
      t.push({message:message.data})
      setChat([...t]);
      console.log([...t])
      // fetchMessages();
    };
  }, []); // Empty dependency array to run only on mount


  const fetchMessages = async () => {
    try {
      const response = await fetch('http://localhost:8080/messages');
      if (!response.ok) {
        throw new Error('Failed to fetch messages');
      }
      const messages = await response.json();
      console.log("6666666666666666")
      setChat(messages);
    } catch (error) {
      console.error(error);
    }
  };

  const handleSubmit = (event:any) => {
    event.preventDefault();
    client.send(message );
    setMessage('');
    // console.log(message)
  };



  return (
<div>
  <input type="text" value={message} onChange={e => setMessage(e.target.value)} />
  <button onClick={handleSubmit}>send</button>
  {chat.map((c,i) => <p key={i}>{c.message}</p>)}
</div>
  )


}

export default Chat;
