import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class WebsocketService {
  private socket: WebSocket | null = null;
  private messagesSubject$ = new Subject<any>();
  public messages$ = this.messagesSubject$.asObservable();

  connect(token: string): void {
    const wsUrl = `ws://localhost:8091/api/ws?websocket_token=${token}`;
    console.log("Connecting to:", wsUrl);

    this.socket = new WebSocket(wsUrl);

    this.socket.onopen = () => {
      console.log('WebSocket connection opened');
    };

    this.socket.onerror = (e) => {
      console.error('WebSocket error:', e);
    };

    this.socket.onclose = (e) => {
      console.warn('WebSocket closed:', e.code, e.reason);
    };

    this.socket.onmessage = (msg) => {
      try {
        const data = JSON.parse(msg.data);
        this.messagesSubject$.next(data);
      } catch (e) {
        console.error("Failed to parse WebSocket message:", msg.data);
      }
    };
  }

  sendMessage(msg: any): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(msg));
    } else {
      console.warn("WebSocket is not open. Message not sent.");
    }
  }

  close(): void {
    this.socket?.close();
  }
}
