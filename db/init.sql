-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants (Companies, MSPs, or Agencies)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    plan_tier VARCHAR(50) DEFAULT 'free', -- free, smb, msp, enterprise
    stripe_customer_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Users (Belonging to Tenants)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) DEFAULT 'admin', -- owner, admin, read-only
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Client Workspaces (Specifically for MSPs/Agencies to isolate their client assets)
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Certificates
CREATE TABLE certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    common_name VARCHAR(255) NOT NULL,
    subject_alternative_names TEXT[],
    issuer_organization VARCHAR(255),
    serial_number VARCHAR(128),
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_to TIMESTAMP WITH TIME ZONE NOT NULL,
    signature_algorithm VARCHAR(100),
    key_algorithm VARCHAR(50), -- RSA, ECDSA
    key_size INTEGER, -- 2048, 4096, 256
    chain_valid BOOLEAN DEFAULT TRUE,
    raw_pem TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Monitored Domains / Endpoints
CREATE TABLE monitored_endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    domain_name VARCHAR(255) NOT NULL,
    port INTEGER DEFAULT 443,
    active_certificate_id UUID REFERENCES certificates(id) ON DELETE SET NULL,
    last_scan_status VARCHAR(50), -- healthy, expiring, error, unreachable
    last_scan_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- DNS Delegation Keys (For ACME DNS-01 Challenge Verification)
CREATE TABLE dns_delegation_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    subdomain_prefix VARCHAR(100) UNIQUE NOT NULL, -- uuid-based sub-subdomain
    txt_record_value VARCHAR(255), -- holds current challenge token
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert a default tenant and workspace for local development / testing
INSERT INTO tenants (id, name, plan_tier) VALUES 
('d5089308-25f0-4573-b6d3-c90a1e3557e4', 'Local Dev Corp', 'msp')
ON CONFLICT DO NOTHING;

INSERT INTO workspaces (id, tenant_id, name, description) VALUES
('b27e69f8-b3d9-43c2-84bb-762bc2b55f24', 'd5089308-25f0-4573-b6d3-c90a1e3557e4', 'Development Default Workspace', 'Primary workspace for development tests')
ON CONFLICT DO NOTHING;

INSERT INTO users (tenant_id, email, role) VALUES
('d5089308-25f0-4573-b6d3-c90a1e3557e4', 'dev@certpulse.local', 'owner')
ON CONFLICT DO NOTHING;
