import React, { useState, useEffect } from 'react'
import Head from 'next/head'
import { 
  Shield, 
  Activity, 
  AlertTriangle, 
  CheckCircle2, 
  Plus, 
  RefreshCw, 
  Clock, 
  Layers, 
  Globe, 
  Sliders, 
  LogOut 
} from 'lucide-react'

export default function Home() {
  const [endpoints, setEndpoints] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddModal, setShowAddModal] = useState(false)
  const [newDomain, setNewDomain] = useState('')
  const [newPort, setNewPort] = useState(443)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [scanningId, setScanningId] = useState(null)

  // Fetch monitored endpoints
  const fetchEndpoints = async () => {
    try {
      setLoading(true)
      const res = await fetch('http://localhost:5000/api/endpoints')
      if (res.ok) {
        const data = await res.json()
        setEndpoints(data)
      }
    } catch (err) {
      console.error("Failed to fetch endpoints:", err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchEndpoints()
  }, [])

  // Handle adding a new domain
  const handleAddDomain = async (e) => {
    e.preventDefault()
    if (!newDomain) return
    setIsSubmitting(true)

    try {
      const res = await fetch('http://localhost:5000/api/endpoints', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          domain_name: newDomain,
          port: Number(newPort)
        })
      })
      if (res.ok) {
        setNewDomain('')
        setNewPort(443)
        setShowAddModal(false)
        fetchEndpoints()
      }
    } catch (err) {
      console.error("Failed to add domain:", err)
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle triggering an instant scan
  const handleTriggerScan = async (id) => {
    setScanningId(id)
    try {
      const res = await fetch(`http://localhost:5000/api/endpoints/${id}/scan`, {
        method: 'POST'
      })
      if (res.ok) {
        fetchEndpoints()
      }
    } catch (err) {
      console.error("Failed to trigger scan:", err)
    } finally {
      setScanningId(null)
    }
  }

  // Derived statistics
  const totalCerts = endpoints.length
  const expiringCount = endpoints.filter(e => e.last_scan_status === 'expiring').length
  const expiredCount = endpoints.filter(e => e.last_scan_status === 'expired' || e.last_scan_status === 'unreachable').length
  const healthyCount = endpoints.filter(e => e.last_scan_status === 'healthy').length
  const healthRate = totalCerts > 0 ? Math.round((healthyCount / totalCerts) * 100) : 100

  // Format Expiry Date
  const formatExpiry = (dateStr) => {
    if (!dateStr) return 'N/A'
    const date = new Date(dateStr)
    const daysLeft = Math.round((date - new Date()) / (1000 * 60 * 60 * 24))
    if (daysLeft < 0) return 'Expired'
    return `${daysLeft} days left`
  }

  return (
    <div className="app-container">
      <Head>
        <title>CertPulse - SSL/TLS Certificate Lifecycle Management</title>
        <meta name="description" content="Next-Generation CLM Dashboard for the Shorter Validity Era" />
        <link rel="icon" href="/favicon.ico" />
      </Head>

      {/* Sidebar Navigation */}
      <aside className="sidebar">
        <div className="logo-container">
          <Shield size={26} color="hsl(var(--primary))" />
          <span>CertPulse</span>
        </div>

        <nav>
          <ul className="nav-links">
            <li>
              <a className="nav-item active">
                <Activity size={18} />
                Overview
              </a>
            </li>
            <li>
              <a className="nav-item">
                <Layers size={18} />
                Certificates
              </a>
            </li>
            <li>
              <a className="nav-item">
                <Clock size={18} />
                Automations
              </a>
            </li>
            <li>
              <a className="nav-item">
                <Globe size={18} />
                Integrations
              </a>
            </li>
            <li>
              <a className="nav-item">
                <Sliders size={18} />
                Settings
              </a>
            </li>
          </ul>
        </nav>
      </aside>

      {/* Main Dashboard Section */}
      <main className="main-content">
        <header className="header-section">
          <div className="header-title">
            <h1>Certificate Command Center</h1>
            <p>Monitor, discover, and automate SSL/TLS certificates across your multi-cloud environment.</p>
          </div>
          <button className="btn-primary" onClick={() => setShowAddModal(true)}>
            <Plus size={18} />
            Monitor Domain
          </button>
        </header>

        {/* Stats Section */}
        <section className="stats-grid">
          <div className="stat-card primary">
            <div className="stat-header">
              <span>TOTAL CERTIFICATES</span>
              <Layers size={18} />
            </div>
            <div className="stat-value">{totalCerts}</div>
          </div>

          <div className="stat-card success">
            <div className="stat-header">
              <span>HEALTH RATE</span>
              <CheckCircle2 size={18} />
            </div>
            <div className="stat-value">{healthRate}%</div>
          </div>

          <div className="stat-card warning">
            <div className="stat-header">
              <span>EXPIRING SOON</span>
              <Clock size={18} />
            </div>
            <div className="stat-value">{expiringCount}</div>
          </div>

          <div className="stat-card danger">
            <div className="stat-header">
              <span>ALERTS / EXPIRED</span>
              <AlertTriangle size={18} />
            </div>
            <div className="stat-value">{expiredCount}</div>
          </div>
        </section>

        {/* Domains List Card */}
        <section className="table-card">
          <div className="table-header">
            <h2>Active Certificate Inventory</h2>
            <button className="btn-secondary" style={{ padding: '0.4rem 0.8rem', display: 'flex', alignItems: 'center', gap: '0.4rem' }} onClick={fetchEndpoints}>
              <RefreshCw size={14} />
              Reload
            </button>
          </div>

          {loading ? (
            <div style={{ padding: '3rem', textAlign: 'center', color: 'hsl(var(--text-muted))' }}>
              <RefreshCw size={24} className="animate-spin" style={{ animation: 'spin 1s linear infinite' }} />
              <p style={{ marginTop: '1rem' }}>Querying certificates database...</p>
            </div>
          ) : endpoints.length === 0 ? (
            <div style={{ padding: '4rem', textAlign: 'center', color: 'hsl(var(--text-muted))' }}>
              <Shield size={48} style={{ opacity: 0.2, marginBottom: '1rem' }} />
              <p>No domains are currently being monitored. Add your first domain above!</p>
            </div>
          ) : (
            <table className="clm-table">
              <thead>
                <tr>
                  <th>Common Name</th>
                  <th>Target Domain</th>
                  <th>Issuer</th>
                  <th>Status</th>
                  <th>Remaining Lifetime</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {endpoints.map((item) => (
                  <tr key={item.id}>
                    <td style={{ fontWeight: '600' }}>{item.common_name || 'Scanning...'}</td>
                    <td>
                      <code style={{ fontSize: '0.85rem', opacity: 0.85 }}>
                        {item.domain_name}:{item.port}
                      </code>
                    </td>
                    <td>{item.issuer_organization || '-'}</td>
                    <td>
                      <span className={`status-badge ${item.last_scan_status}`}>
                        {item.last_scan_status === 'healthy' && <CheckCircle2 size={12} />}
                        {item.last_scan_status === 'expiring' && <Clock size={12} />}
                        {(item.last_scan_status === 'expired' || item.last_scan_status === 'unreachable') && <AlertTriangle size={12} />}
                        {item.last_scan_status}
                      </span>
                    </td>
                    <td style={{ color: item.last_scan_status === 'expiring' ? 'hsl(var(--warning))' : item.last_scan_status === 'expired' ? 'hsl(var(--danger))' : 'inherit' }}>
                      {formatExpiry(item.valid_to)}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <button 
                        className="btn-secondary" 
                        style={{ padding: '0.35rem 0.75rem', fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
                        onClick={() => handleTriggerScan(item.id)}
                        disabled={scanningId === item.id}
                      >
                        <RefreshCw size={12} className={scanningId === item.id ? 'animate-spin' : ''} style={{ animation: scanningId === item.id ? 'spin 1s linear infinite' : 'none' }} />
                        {scanningId === item.id ? 'Scanning' : 'Scan Now'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </main>

      {/* Add Domain Modal */}
      {showAddModal && (
        <div className="modal-overlay">
          <div className="modal-content">
            <h3 className="modal-title">Monitor New Endpoint</h3>
            <form onSubmit={handleAddDomain}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
                <div className="form-group">
                  <label htmlFor="domain">Domain Name</label>
                  <input 
                    type="text" 
                    id="domain" 
                    className="form-input" 
                    placeholder="e.g., example.com"
                    value={newDomain}
                    onChange={(e) => setNewDomain(e.target.value)}
                    required 
                  />
                </div>
                <div className="form-group">
                  <label htmlFor="port">Port</label>
                  <input 
                    type="number" 
                    id="port" 
                    className="form-input" 
                    value={newPort}
                    onChange={(e) => setNewPort(e.target.value)}
                    required 
                  />
                </div>
                <div className="modal-actions">
                  <button type="button" className="btn-secondary" onClick={() => setShowAddModal(false)} disabled={isSubmitting}>
                    Cancel
                  </button>
                  <button type="submit" className="btn-primary" disabled={isSubmitting}>
                    {isSubmitting ? 'Registering...' : 'Add Monitor'}
                  </button>
                </div>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Keyframe animation for loaders */}
      <style jsx global>{`
        @keyframes spin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  )
}
