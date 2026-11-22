import React from 'react';

interface State {
  error: Error | null;
}

export class ErrorBoundary extends React.Component<{ children: React.ReactNode }, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error) {
    console.error('[openmimo] UI crash caught:', error);
  }

  render() {
    if (this.state.error) {
      return (
        <div
          style={{
            height: '100vh',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 24,
            background: '#f4f6f9',
            fontFamily: '-apple-system, BlinkMacSystemFont, Inter, sans-serif',
          }}
        >
          <div
            style={{
              maxWidth: 520,
              width: '100%',
              background: '#fff',
              border: '1px solid #fecaca',
              borderRadius: 12,
              padding: 22,
              boxShadow: '0 8px 24px rgba(15,23,42,0.08)',
            }}
          >
            <h2 style={{ margin: '0 0 8px', fontSize: 16, color: '#b91c1c' }}>
              Ha ocurrido un error en la interfaz
            </h2>
            <p style={{ margin: '0 0 12px', fontSize: 13, color: '#475569' }}>
              {this.state.error.message || 'Error inesperado'}
            </p>
            <pre
              style={{
                background: '#f8fafc',
                border: '1px solid #e2e8f0',
                borderRadius: 6,
                padding: 10,
                fontSize: 11,
                maxHeight: 160,
                overflow: 'auto',
                color: '#334155',
              }}
            >
              {this.state.error.stack || String(this.state.error)}
            </pre>
            <div style={{ display: 'flex', gap: 8, marginTop: 14 }}>
              <button
                className="btn btn-primary"
                onClick={() => {
                  this.setState({ error: null });
                  window.location.reload();
                }}
              >
                Recargar openmimo
              </button>
            </div>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
