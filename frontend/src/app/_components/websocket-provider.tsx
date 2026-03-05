"use client";

import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
} from "react";
import useWebSocket, { ReadyState } from "react-use-websocket";
import { GET_WS_URL } from "@/lib/api";

interface WebSocketMessage {
  topic: string;
  payload: any;
}

type SubscriptionCallback = (payload: any) => void;

interface WebSocketContextType {
  subscribe: (topic: string, callback: SubscriptionCallback) => void;
  unsubscribe: (topic: string, callback: SubscriptionCallback) => void;
  readyState: ReadyState;
}

const WebSocketContext = createContext<WebSocketContextType | null>(null);

export const useWebSocketContext = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error(
      "useWebSocketContext must be used within a WebSocketProvider",
    );
  }
  return context;
};

export const WebSocketProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const { sendMessage, lastMessage, readyState } = useWebSocket(GET_WS_URL(), {
    share: true,
    shouldReconnect: () => true,
  });
  const subscriptions = useRef<Map<string, Set<SubscriptionCallback>>>(
    new Map(),
  );

  useEffect(() => {
    if (lastMessage !== null) {
      try {
        const message = JSON.parse(lastMessage.data) as WebSocketMessage;
        const { topic, payload } = message;
        if (subscriptions.current.has(topic)) {
          subscriptions.current.get(topic)?.forEach((callback) => {
            callback(payload);
          });
        }
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    }
  }, [lastMessage]);

  const subscribe = useCallback(
    (topic: string, callback: SubscriptionCallback) => {
      if (!subscriptions.current.has(topic)) {
        subscriptions.current.set(topic, new Set());
      }
      subscriptions.current.get(topic)?.add(callback);
      sendMessage(JSON.stringify({ type: "subscribe", topic }));
    },
    [sendMessage],
  );

  const unsubscribe = useCallback(
    (topic: string, callback: SubscriptionCallback) => {
      if (subscriptions.current.has(topic)) {
        subscriptions.current.get(topic)?.delete(callback);
        if (subscriptions.current.get(topic)?.size === 0) {
          subscriptions.current.delete(topic);
          sendMessage(JSON.stringify({ type: "unsubscribe", topic }));
        }
      }
    },
    [sendMessage],
  );

  const contextValue = {
    subscribe,
    unsubscribe,
    readyState,
  };

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
};
