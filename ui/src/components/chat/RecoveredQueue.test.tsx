import React from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ChatPanel } from './ChatPanel';
import { useAgentStore } from '../../store/agentStore';
import { useHeldStore } from '../../store/heldStore';

const mocks = vi.hoisted(() => ({
  getHeldPrompt: vi.fn(),
  withdrawPrompt: vi.fn(),
  steerPrompt: vi.fn(),
  dismissPromptRecovery: vi.fn(),
}));
vi.mock('../../api/client', async (original) => ({
  ...await original<typeof import('../../api/client')>(),
  getHeldPrompt: mocks.getHeldPrompt,
  withdrawPrompt: mocks.withdrawPrompt,
  steerPrompt: mocks.steerPrompt,
  dismissPromptRecovery: mocks.dismissPromptRecovery,
}));
vi.mock('../../api/sse', () => ({
  sseClient: { registerOpenAgent: vi.fn(() => vi.fn()) },
}));
vi.mock('./TerminalTab', () => ({
  TerminalTab: () => null,
}));
vi.mock('../../api/config', async (original) => ({
  ...await original<typeof import('../../api/config')>(),
  useBackends: () => ({ data: undefined }),
  useProjects: () => ({ data: undefined }),
  useRoles: () => ({ data: undefined }),
}));
afterEach(() => { cleanup(); localStorage.clear(); vi.clearAllMocks(); });

// RECOVERED-QUEUE-1 (FS-03.R90/R100): failed against the pre-fix implementation.
it('restores withdrawn text to the draft without subsequently offering a second recovered copy', async () => {
  const agent = {
    agent_id: 'repro-agent', name: 'Nova', role: 'implementer', project: 'app',
    backend: 'claude', model: 'sonnet', interface: 'chat', running: true,
    state: 'busy', context_pct: 12, created_at: '2026-11-05T00:00:00Z',
  };
  useAgentStore.setState({ agents: { 'repro-agent': agent as never }, order: ['repro-agent'], hydrated: true });
  useHeldStore.setState({ byAgent: {}, afterSeqByAgent: {}, recoveryByAgent: {}, attachmentsByAgent: {}, recoveryUnreadByAgent: {} });
  const message = { held_id: 'hold-1', request_id: 'receipt-1', text: 'inspect the tests', attachments: [] };
  mocks.getHeldPrompt.mockResolvedValue({ agent_id: 'repro-agent', ...message, after_seq: 1 });
  mocks.withdrawPrompt.mockResolvedValue({ withdrawn: true, message });
  mocks.dismissPromptRecovery.mockImplementation(async () => {
    mocks.getHeldPrompt.mockResolvedValue({ agent_id: 'repro-agent', text: '', after_seq: 0 });
  });
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <MemoryRouter initialEntries={['/agent/repro-agent']}><Routes><Route path='/agent/:id' element={<ChatPanel />} /></Routes></MemoryRouter>
  </QueryClientProvider>);
  const withdraw = await screen.findByRole('button', { name: 'Withdraw queued' });
  fireEvent.click(withdraw);
  await waitFor(() => expect((screen.getByRole('textbox') as HTMLTextAreaElement).value).toBe(message.text));

  // The exact released receipt is acknowledged after durable draft restoration.
  expect(mocks.dismissPromptRecovery).toHaveBeenCalledWith('repro-agent', message.request_id);
  act(() => useAgentStore.setState({ agents: { 'repro-agent': { ...agent, state: 'idle' } as never } }));
  await waitFor(() => expect(mocks.getHeldPrompt).toHaveBeenCalledTimes(3));
  expect((screen.getByRole('textbox') as HTMLTextAreaElement).value).toBe(message.text);
  expect(JSON.parse(localStorage.getItem('chuck-chat-drafts')!).drafts['repro-agent'].text).toBe(message.text);
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Send recovered message' })).toBeNull());
});

// RECOVERED-QUEUE-2 (FS-03.R50/R100): the real delayed-ack backend fixture also
// confirms unknown is exposed during Steer and disappears after success.
it('clears a transient recovery snapshot once the matching Steer succeeds', async () => {
  const agent = {
    agent_id: 'steer-agent', name: 'Nova', role: 'implementer', project: 'app',
    backend: 'claude', model: 'sonnet', interface: 'chat', running: true,
    state: 'busy', steering_available: true, context_pct: 12, created_at: '2026-11-05T00:00:00Z',
  };
  useAgentStore.setState({ agents: { 'steer-agent': agent as never }, order: ['steer-agent'], hydrated: true });
  useHeldStore.setState({ byAgent: {}, afterSeqByAgent: {}, recoveryByAgent: {}, attachmentsByAgent: {}, recoveryUnreadByAgent: {} });
  const message = { held_id: 'steer-hold', request_id: 'steer-receipt', text: 'inspect the tests', attachments: [] };
  mocks.getHeldPrompt.mockResolvedValue({ agent_id: 'steer-agent', ...message, after_seq: 1 });
  let completeSteer!: (value: unknown) => void;
  mocks.steerPrompt.mockImplementation(() => new Promise((resolve) => { completeSteer = resolve; }));
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <MemoryRouter initialEntries={['/agent/steer-agent']}><Routes><Route path='/agent/:id' element={<ChatPanel />} /></Routes></MemoryRouter>
  </QueryClientProvider>);
  await screen.findByRole('button', { name: 'Withdraw queued' });
  fireEvent.click(screen.getByRole('button', { name: 'Steer' }));
  await waitFor(() => expect(mocks.steerPrompt).toHaveBeenCalled());
  // A state-triggered read occurs while the adapter call is outstanding.
  mocks.getHeldPrompt.mockResolvedValue({ agent_id: 'steer-agent', text: '', after_seq: 0,
    recovery: { request_id: message.request_id, text: message.text, attachments: [], delivery: 'unknown' } });
  act(() => useAgentStore.setState({ agents: { 'steer-agent': { ...agent, state: 'idle' } as never } }));
  await waitFor(() => expect(mocks.getHeldPrompt).toHaveBeenCalledTimes(2));
  // The server would now report submitted, but successful Steer never reads it.
  mocks.getHeldPrompt.mockResolvedValue({ agent_id: 'steer-agent', text: '', after_seq: 0 });
  await act(async () => completeSteer({ accepted: true, agent_id: 'steer-agent', outcome: 'steered' }));
  await screen.findByText('Delivered into the running turn.');
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Send recovered message' })).toBeNull());
});
