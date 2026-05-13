import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';

const fetchMock = vi.fn();

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe('App', () => {
  it('falls back to bundled pricing and shows an accessible error state', async () => {
    fetchMock.mockRejectedValue(new Error('network down'));

    render(<App />);

    expect(screen.getByRole('status')).toHaveTextContent(/loading live pricing/i);
    expect(await screen.findByRole('alert')).toHaveTextContent(/using bundled pricing snapshot/i);
    expect(screen.getByRole('heading', { name: /healthtech ingest cost calculator/i })).toBeInTheDocument();
    expect(screen.getByRole('spinbutton', { name: /^dau$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /export pdf/i })).toBeInTheDocument();
  });

  it('recomputes the per-user display when DAU changes', async () => {
    fetchMock.mockRejectedValue(new Error('offline'));
    render(<App />);
    await screen.findByRole('alert');

    const perUser = screen.getByLabelText(/per user monthly cost/i);
    expect(perUser).toHaveClass('per-user-green');

    fireEvent.change(screen.getByRole('spinbutton', { name: /^dau$/i }), { target: { value: '100' } });

    await waitFor(() => expect(screen.getByLabelText(/per user monthly cost/i)).toHaveTextContent(/\$/));
    expect(screen.getByText(/total monthly estimate/i)).toBeInTheDocument();
  });

  it('lets scenario toggles recompute meaningful deltas', async () => {
    fetchMock.mockRejectedValue(new Error('offline'));
    render(<App />);
    await screen.findByRole('alert');

    expect(screen.getByText(/no tiering/i).nextElementSibling?.textContent).toMatch(/\$[1-9]/);
    fireEvent.click(screen.getByLabelText(/with s3 lifecycle tiering/i));

    expect(screen.getByLabelText(/with s3 lifecycle tiering/i)).not.toBeChecked();
  });
});
