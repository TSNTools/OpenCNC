import { useEffect, useMemo, useState } from "react";
import { addStream, deleteStream, loadStreamNodes, loadStreams, updateStream } from "./api";
import { Panel, PanelHeader } from "./components";
import type { NodeOption, StreamFormState, StreamRecord, StreamRankValue, TrafficTypeValue } from "./types";

const trafficTypeOptions: Array<{ value: TrafficTypeValue; label: string }> = [
  { value: "TRAFFIC_TYPE_ISOCHRONOUS", label: "Isochronous" },
  { value: "TRAFFIC_TYPE_SYNCHRONOUS", label: "Synchronous" },
  { value: "TRAFFIC_TYPE_ASYNCHRONOUS", label: "Asynchronous" },
  { value: "TRAFFIC_TYPE_MANAGEMENT", label: "Management" },
  { value: "TRAFFIC_TYPE_ALARM", label: "Alarm" },
  { value: "TRAFFIC_TYPE_BEST_EFFORT_HIGH", label: "Best effort high" },
  { value: "TRAFFIC_TYPE_BEST_EFFORT_LOW", label: "Best effort low" },
];

const rankOptions: Array<{ value: StreamRankValue; label: string }> = [
  { value: "RANK_UNSPECIFIED", label: "Unspecified" },
  { value: "RANK_A", label: "Rank A" },
  { value: "RANK_B", label: "Rank B" },
];

function defaultForm(nodes: NodeOption[]): StreamFormState {
  const firstNode = nodes[0]?.id ?? "";
  const secondNode = nodes[1]?.id ?? firstNode;

  return {
    name: "",
    talkerNodeId: firstNode,
    listenerNodeIds: secondNode && secondNode !== firstNode ? [secondNode] : [],
    trafficType: "TRAFFIC_TYPE_ISOCHRONOUS",
    rank: "RANK_A",
    destinationMac: "",
    sourceMac: "",
    vlanId: 1,
    intervalNs: 1000000,
    maxFrameSize: 1500,
    maxFramesPerInterval: 1,
    maxLatencyNs: 1000000,
    maxJitterNs: 100000,
    minTransmitOffsetNs: 0,
    maxTransmitOffsetNs: 0,
    numSeamlessTrees: 1,
  };
}

function recordToForm(stream: StreamRecord): StreamFormState {
  return {
    name: stream.name,
    talkerNodeId: stream.talkerNodeId || stream.source,
    listenerNodeIds: stream.listenerNodeIds,
    trafficType: stream.trafficType,
    rank: stream.rank,
    destinationMac: stream.destinationMac,
    sourceMac: stream.sourceMac,
    vlanId: stream.vlanId,
    intervalNs: stream.intervalNs,
    maxFrameSize: stream.maxFrameSize,
    maxFramesPerInterval: stream.maxFramesPerInterval,
    maxLatencyNs: stream.maxLatencyNs,
    maxJitterNs: stream.maxJitterNs,
    minTransmitOffsetNs: stream.minTransmitOffsetNs,
    maxTransmitOffsetNs: stream.maxTransmitOffsetNs,
    numSeamlessTrees: stream.numSeamlessTrees,
  };
}

function splitCharacteristics(characteristics: string) {
  const lines = characteristics
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  return lines.length > 0 ? lines : ["-"];
}

export function StreamsScreen({ refreshToken }: { refreshToken: number }) {
  const [nodes, setNodes] = useState<NodeOption[]>([]);
  const [streams, setStreams] = useState<StreamRecord[]>([]);
  const [selectedStreamId, setSelectedStreamId] = useState("");
  const [form, setForm] = useState<StreamFormState>(defaultForm([]));
  const [statusMessage, setStatusMessage] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  useEffect(() => {
    let active = true;

    async function load() {
      setBusy(true);
      setErrorMessage("");

      try {
        const [loadedNodes, loadedStreams] = await Promise.all([loadStreamNodes(), loadStreams()]);
        if (!active) {
          return;
        }

        setNodes(loadedNodes);
        setStreams(loadedStreams);

        const selected = loadedStreams.find((stream) => stream.id === selectedStreamId) ?? loadedStreams[0];
        if (selected) {
          setSelectedStreamId(selected.id);
          setForm(recordToForm(selected));
        } else {
          setSelectedStreamId("");
          setForm(defaultForm(loadedNodes));
        }
      } catch (error) {
        if (active) {
          setErrorMessage(error instanceof Error ? error.message : "Failed to load streams");
        }
      } finally {
        if (active) {
          setBusy(false);
        }
      }
    }

    void load();

    return () => {
      active = false;
    };
  }, [refreshToken]);

  useEffect(() => {
    if (!selectedStreamId) {
      return;
    }

    const selected = streams.find((stream) => stream.id === selectedStreamId);
    if (selected) {
      setForm(recordToForm(selected));
    }
  }, [selectedStreamId, streams]);

  const selectedStream = useMemo(
    () => streams.find((stream) => stream.id === selectedStreamId),
    [selectedStreamId, streams],
  );

  function updateField<K extends keyof StreamFormState>(key: K, value: StreamFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function updateNumberField(
    key: keyof Pick<
      StreamFormState,
      | "vlanId"
      | "intervalNs"
      | "maxFrameSize"
      | "maxFramesPerInterval"
      | "maxLatencyNs"
      | "maxJitterNs"
      | "minTransmitOffsetNs"
      | "maxTransmitOffsetNs"
      | "numSeamlessTrees"
    >,
    value: string,
  ) {
    const parsed = Number(value);
    setForm((current) => ({ ...current, [key]: Number.isFinite(parsed) ? parsed : 0 }));
  }

  async function reloadStreams() {
    const loadedStreams = await loadStreams();
    setStreams(loadedStreams);

    const selected = loadedStreams.find((stream) => stream.id === selectedStreamId) ?? loadedStreams[0];
    if (selected) {
      setSelectedStreamId(selected.id);
      setForm(recordToForm(selected));
    }
  }

  async function handleAddStream() {
    setBusy(true);
    setErrorMessage("");
    try {
      await addStream({ ...form, listenerNodeIds: [...form.listenerNodeIds] });
      await reloadStreams();
      setStatusMessage("Stream added");
      setSelectedFile(null);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to add stream");
    } finally {
      setBusy(false);
    }
  }

  async function handleUpdateStream() {
    if (!selectedStreamId) {
      setErrorMessage("Select a stream first");
      return;
    }

    setBusy(true);
    setErrorMessage("");
    try {
      await updateStream(selectedStreamId, { ...form, listenerNodeIds: [...form.listenerNodeIds] });
      await reloadStreams();
      setStatusMessage("Stream updated");
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to update stream");
    } finally {
      setBusy(false);
    }
  }

  async function handleDeleteStream() {
    if (!selectedStreamId) {
      setErrorMessage("Select a stream first");
      return;
    }

    setBusy(true);
    setErrorMessage("");
    try {
      await deleteStream(selectedStreamId);
      await reloadStreams();
      setStatusMessage("Stream deleted");
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to delete stream");
    } finally {
      setBusy(false);
    }
  }

  async function handleImportStream() {
    if (!selectedFile) {
      setErrorMessage("Choose a file first");
      return;
    }

    setBusy(true);
    setErrorMessage("");
    try {
      const parsed = JSON.parse(await selectedFile.text()) as Partial<StreamFormState>;
      const nextForm: StreamFormState = {
        ...defaultForm(nodes),
        ...parsed,
        listenerNodeIds: Array.isArray(parsed.listenerNodeIds) ? parsed.listenerNodeIds : form.listenerNodeIds,
      };

      await addStream(nextForm);
      await reloadStreams();
      setStatusMessage("Stream imported");
      setSelectedFile(null);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to import stream");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="streams-page">
      <Panel className="stream-form-panel">
        <div className="streams-toolbar">
          <label className="file-picker">
            <span>Choose File</span>
            <input type="file" accept="application/json,.json,.txt" onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)} />
          </label>
          <button className="primary-button stream-import-button" type="button" onClick={handleImportStream} disabled={busy}>
            Import Stream
          </button>
        </div>

        <div className="stream-form-scroll">
          <div className="form-section-header">
            <h3>Create Stream</h3>
            <p>{errorMessage || statusMessage || "Fill out the stream definition and sync it to the backend."}</p>
          </div>

          <div className="form-grid stream-form-grid">
            <label>
              <span>Name</span>
              <input value={form.name} onChange={(event) => updateField("name", event.target.value)} placeholder="Str 5" />
            </label>

            <div className="form-group">
              <span className="group-label">Data Frame</span>
              <div className="field-grid split">
                <label>
                  <span>Talker</span>
                  <select value={form.talkerNodeId} onChange={(event) => updateField("talkerNodeId", event.target.value)}>
                    {nodes.map((node) => (
                      <option key={node.id} value={node.id}>
                        {node.label}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  <span>Listeners</span>
                  <select
                    multiple
                    size={Math.min(Math.max(nodes.length, 2), 4)}
                    value={form.listenerNodeIds}
                    onChange={(event) => updateField("listenerNodeIds", Array.from(event.target.selectedOptions).map((option) => option.value))}
                  >
                    {nodes.map((node) => (
                      <option key={node.id} value={node.id}>
                        {node.label}
                      </option>
                    ))}
                  </select>
                </label>
              </div>

              <label>
                <span>VLAN ID</span>
                <input type="number" min="0" value={form.vlanId} onChange={(event) => updateNumberField("vlanId", event.target.value)} />
              </label>
            </div>

            <div className="form-group">
              <span className="group-label">Traffic Class</span>
              <div className="field-grid split">
                <label>
                  <span>Traffic Type</span>
                  <select value={form.trafficType} onChange={(event) => updateField("trafficType", event.target.value as TrafficTypeValue)}>
                    {trafficTypeOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  <span>Stream Rank</span>
                  <select value={form.rank} onChange={(event) => updateField("rank", event.target.value as StreamRankValue)}>
                    {rankOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            </div>

            <div className="form-group">
              <span className="group-label">Traffic Specification</span>
              <div className="field-grid triple">
                <label>
                  <span>Interval (ns)</span>
                  <input type="number" min="0" value={form.intervalNs} onChange={(event) => updateNumberField("intervalNs", event.target.value)} />
                </label>
                <label>
                  <span>Max Frame Size</span>
                  <input type="number" min="0" value={form.maxFrameSize} onChange={(event) => updateNumberField("maxFrameSize", event.target.value)} />
                </label>
                <label>
                  <span>Max Frames per interval</span>
                  <input type="number" min="0" value={form.maxFramesPerInterval} onChange={(event) => updateNumberField("maxFramesPerInterval", event.target.value)} />
                </label>
              </div>
            </div>

            <div className="form-group">
              <span className="group-label">User-to-Network Requirements</span>
              <div className="field-grid double">
                <label>
                  <span>Max Latency (ns)</span>
                  <input type="number" min="0" value={form.maxLatencyNs} onChange={(event) => updateNumberField("maxLatencyNs", event.target.value)} />
                </label>
                <label>
                  <span>Max Jitter (ns)</span>
                  <input type="number" min="0" value={form.maxJitterNs} onChange={(event) => updateNumberField("maxJitterNs", event.target.value)} />
                </label>
                <label>
                  <span>Minimum Transmit Offset (ns)</span>
                  <input type="number" min="0" value={form.minTransmitOffsetNs} onChange={(event) => updateNumberField("minTransmitOffsetNs", event.target.value)} />
                </label>
                <label>
                  <span>Maximum Transmit Offset (ns)</span>
                  <input type="number" min="0" value={form.maxTransmitOffsetNs} onChange={(event) => updateNumberField("maxTransmitOffsetNs", event.target.value)} />
                </label>
                <label>
                  <span>Number of Seamless Trees</span>
                  <input type="number" min="0" value={form.numSeamlessTrees} onChange={(event) => updateNumberField("numSeamlessTrees", event.target.value)} />
                </label>
              </div>
            </div>
          </div>

          <div className="action-row stream-action-row">
            <button className="primary-button" type="button" onClick={handleAddStream} disabled={busy}>
              Add Stream
            </button>
          </div>
        </div>
      </Panel>

      <Panel className="stream-list-panel">
        <PanelHeader title="Available Streams" />
        <div className="stream-list" role="listbox" aria-label="Available streams">
          {streams.map((stream) => (
            <button
              key={stream.id}
              type="button"
              className={stream.id === selectedStreamId ? "stream-list-item is-selected" : "stream-list-item"}
              onClick={() => {
                setSelectedStreamId(stream.id);
                setForm(recordToForm(stream));
              }}
            >
              <strong>{stream.name}</strong>
              <span>{stream.source || "No source"}</span>
            </button>
          ))}
          {streams.length === 0 ? <p className="empty-list-note">No streams available</p> : null}
        </div>
      </Panel>

      <Panel className="stream-details-panel">
        <PanelHeader title="Stream details" />
        {selectedStream ? (
          <dl className="stream-details-grid">
            <div>
              <dt>Name</dt>
              <dd>{selectedStream.name}</dd>
            </div>
            <div>
              <dt>Source</dt>
              <dd>{selectedStream.source || "-"}</dd>
            </div>
            <div>
              <dt>Listeners</dt>
              <dd>{selectedStream.listeners || "-"}</dd>
            </div>
            <div>
              <dt>Characteristics</dt>
              <dd className="stream-characteristics">
                {splitCharacteristics(selectedStream.characteristics).map((line) => (
                  <span key={line}>{line}</span>
                ))}
              </dd>
            </div>
          </dl>
        ) : (
          <div className="stream-details-empty">
            <p>Select a stream to view details.</p>
          </div>
        )}

        <div className="stream-details-actions">
          <button className="ghost-button" type="button" onClick={handleUpdateStream} disabled={busy || !selectedStreamId}>
            Edit Stream
          </button>
          <button className="danger-button" type="button" onClick={handleDeleteStream} disabled={busy || !selectedStreamId}>
            Delete Stream
          </button>
        </div>
      </Panel>
    </div>
  );
}