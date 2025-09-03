# Domain Transfer: logans3dcreations.com

_Documentation for transferring domain from Square to DNSimple_

## Current DNS Configuration

### Current Domain Registrar
- **Registrar**: Square (Web.com / register.com infrastructure)
- **Domain**: logans3dcreations.com
- **Status**: Active

### Current Nameservers
```
dns046.a.register.com
dns083.d.register.com  
dns041.c.register.com
dns192.b.register.com
```

### Current DNS Records

#### A Records
| Subdomain | Type | Value | TTL |
|-----------|------|--------|-----|
| @ (root) | A | 199.34.228.159 | 3600 |
| www | A | 199.34.228.159 | 3600 |
| ftp | A | 199.34.228.159 | 3600 |
| shop | A | 199.34.228.159 | 3600 |
| admin | A | 199.34.228.159 | 3600 |
| blog | A | 199.34.228.159 | 3600 |
| api | A | 199.34.228.159 | 3600 |
| mail | A | 216.21.224.199 | 3600 |

#### MX Records (Google Workspace)
| Priority | Mail Server |
|----------|-------------|
| 1 | aspmx.l.google.com |
| 5 | alt1.aspmx.l.google.com |
| 5 | alt2.aspmx.l.google.com |
| 10 | alt3.aspmx.l.google.com |
| 10 | alt4.aspmx.l.google.com |

#### TXT Records
| Type | Value |
|------|-------|
| TXT | "google-site-verification=4hN30ZRAlwou7ULhGC4Y2-bYMjuNNZwhIlMjwcbHCps" |

#### SOA Record
```
Primary: dns046.a.register.com
Email: root.register.com
Serial: 2023080905
Refresh: 28800
Retry: 7200  
Expire: 604800
Minimum: 3600
```

---

## Pre-Transfer Checklist

### 1. Backup Current Configuration
- ✅ DNS records documented above
- ⚠️  **Action Required**: Screenshot Square's DNS management interface
- ⚠️  **Action Required**: Export any additional settings from Square

### 2. Verify Services
- ✅ Google Workspace email is configured via MX records
- ⚠️  **Action Required**: Confirm all subdomains are actively used
- ⚠️  **Action Required**: Identify any services that might break during transfer

### 3. Prepare DNSimple
- ⚠️  **Action Required**: Ensure DNSimple account is active
- ⚠️  **Action Required**: Pre-configure DNS records in DNSimple (don't activate yet)

---

## Domain Transfer Process

### Phase 1: Preparation (1-2 days before transfer)

#### Step 1: Lower TTL Values
1. In Square's DNS management, reduce TTL for all records to 300 seconds (5 minutes)
2. Wait 24-48 hours for propagation
3. This minimizes downtime during the actual transfer

#### Step 2: Setup DNSimple DNS Records
Pre-configure these records in DNSimple (but don't activate nameservers yet):

```dns
# A Records
@               A       199.34.228.159
www             A       199.34.228.159  
ftp             A       199.34.228.159
shop            A       199.34.228.159
admin           A       199.34.228.159
blog            A       199.34.228.159
api             A       199.34.228.159
mail            A       216.21.224.199

# MX Records (Google Workspace)
@               MX 1    aspmx.l.google.com.
@               MX 5    alt1.aspmx.l.google.com.
@               MX 5    alt2.aspmx.l.google.com.
@               MX 10   alt3.aspmx.l.google.com.
@               MX 10   alt4.aspmx.l.google.com.

# TXT Records
@               TXT     "google-site-verification=4hN30ZRAlwou7ULhGC4Y2-bYMjuNNZwhIlMjwcbHCps"

# Future Vercel Records (to be added after new site is ready)
# @             A       76.76.19.61  # Vercel's IP (example)
# www           CNAME   cname.vercel-dns.com.
```

### Phase 2: Domain Transfer

#### Step 1: Initiate Transfer at Square
1. Log into Square domain management
2. Unlock the domain for transfer
3. Request EPP/Authorization code
4. Disable auto-renewal (if enabled)
5. Ensure domain has at least 60 days before expiration

#### Step 2: Start Transfer at DNSimple
1. Log into DNSimple
2. Initiate domain transfer for `logans3dcreations.com`
3. Enter EPP/Authorization code from Square
4. Confirm transfer details and payment

#### Step 3: Confirm Transfer
1. Check email for transfer confirmation from both registrars
2. Approve transfer request (usually required within 5 days)
3. Monitor transfer status in DNSimple dashboard

### Phase 3: DNS Cutover

#### Step 1: Update Nameservers
Once transfer is complete:
1. In DNSimple, update nameservers to DNSimple's:
   ```
   ns1.dnsimple.com
   ns2.dnsimple.com  
   ns3.dnsimple.com
   ns4.dnsimple.com
   ```

#### Step 2: Verify DNS Propagation
```bash
# Test DNS resolution
dig @8.8.8.8 logans3dcreations.com A
dig @8.8.8.8 logans3dcreations.com MX
dig @8.8.8.8 www.logans3dcreations.com A

# Check multiple DNS servers
dig @1.1.1.1 logans3dcreations.com A
dig @208.67.222.222 logans3dcreations.com A
```

#### Step 3: Test Services
- ✅ Email delivery (send test email)
- ✅ Website accessibility  
- ✅ All subdomains resolve correctly
- ✅ Google Workspace admin console shows domain as verified

---

## Post-Transfer Tasks

### 1. Update DNS for New Architecture

#### For Vercel Deployment
Once the new site is ready for deployment:

```dns
# Remove old hosting records
# Add Vercel records
@               A       76.76.19.61
www             CNAME   cname.vercel-dns.com.

# Keep email and verification records
@               MX 1    aspmx.l.google.com.
# ... (keep all MX records)
@               TXT     "google-site-verification=4hN30ZRAlwou7ULhGC4Y2-bYMjuNNZwhIlMjwcbHCps"

# Add new verification records as needed
@               TXT     "v=spf1 include:_spf.google.com ~all"  # If not already present
```

### 2. Security Enhancements
```dns
# Add security headers (optional)
@               TXT     "v=DMARC1; p=none; rua=mailto:admin@logans3dcreations.com"

# CAA records for SSL certificate authority restrictions
@               CAA     0 issue "letsencrypt.org"
@               CAA     0 issue "digicert.com"  
@               CAA     0 iodef "mailto:admin@logans3dcreations.com"
```

### 3. Monitoring Setup
- Configure DNS monitoring in DNSimple
- Set up uptime monitoring for the new site
- Monitor email delivery for 1-2 weeks

---

## Rollback Plan

If issues arise during transfer:

### Emergency DNS Revert
1. Contact DNSimple support immediately
2. Revert nameservers to original Square nameservers:
   ```
   dns046.a.register.com
   dns083.d.register.com  
   dns041.c.register.com
   dns192.b.register.com
   ```
3. Wait for DNS propagation (5-60 minutes with lowered TTL)

### Service-Specific Issues
- **Email Issues**: Verify MX records are identical to original
- **Website Issues**: Ensure A records point to correct IP addresses
- **SSL Issues**: May need to wait for certificate re-issuance

---

## Timeline Estimate

| Phase | Duration | Tasks |
|-------|----------|-------|
| **Preparation** | 2-3 days | Lower TTLs, setup DNSimple records |
| **Transfer Initiation** | 1 day | Request EPP code, start transfer |
| **Transfer Processing** | 5-7 days | Registrar transfer process |
| **DNS Cutover** | 2-4 hours | Update nameservers, verify propagation |
| **Testing & Validation** | 1-2 days | Comprehensive service testing |

**Total Estimated Time**: 10-17 days

---

## Important Notes

1. **Email Continuity**: Google Workspace email should continue working throughout the transfer as long as MX records are preserved
2. **Website Downtime**: Minimal if DNS records are properly pre-configured
3. **Domain Lock Period**: After transfer, domain will be locked for 60 days (can't transfer again)
4. **Expiration Date**: Transfer may extend expiration by 1 year
5. **Cost**: DNSimple charges for both domain transfer and ongoing annual registration

---

## Emergency Contacts

- **DNSimple Support**: support@dnsimple.com
- **Current Website Host**: [Square support details needed]
- **Email Provider**: Google Workspace admin console

---

_Last Updated: 2025-09-03_