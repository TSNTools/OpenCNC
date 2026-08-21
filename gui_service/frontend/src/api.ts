import type { NodeOption, StreamFormState, StreamRecord } from "./types";

type ApiError = {
  error?: string;
  message?: string;
};

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  const text = await response.text();
  const payload = text ? JSON.parse(text) : null;

  if (!response.ok) {
    const body = payload as ApiError | null;
    throw new Error(body?.error || body?.message || `Request failed with ${response.status}`);
  }

  return payload as T;
}

type BackendNode = {
  id: string;
  name?: string;
  type?: string;
};

export async function loadStreamNodes(): Promise<NodeOption[]> {
  const nodes = await requestJSON<BackendNode[]>("/api/v1/nodes");
  return nodes.map((node) => ({
    id: node.id,
    label: node.name || node.id,
    subtitle: node.type || "Node",
  }));
}

export async function loadStreams(): Promise<StreamRecord[]> {
  return requestJSON<StreamRecord[]>("/api/v1/streams");
}

export async function addStream(payload: StreamFormState): Promise<void> {
  await requestJSON("/api/v1/streams", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function updateStream(streamId: string, payload: StreamFormState): Promise<void> {
  await requestJSON(`/api/v1/streams/${encodeURIComponent(streamId)}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
}

export async function deleteStream(streamId: string): Promise<void> {
  await requestJSON(`/api/v1/streams/${encodeURIComponent(streamId)}`, {
    method: "DELETE",
  });
}