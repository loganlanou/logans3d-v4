#!/bin/bash

# Script to update template signatures to accept PageMeta

# Update about/index.templ
sed -i '' 's/templ Index(c echo.Context) {/@layout.Base(c, layout.PageMeta{/g' views/about/index.templ
if grep -q "templ Index(c echo.Context) {" views/about/index.templ; then
    sed -i '' 's/templ Index(c echo.Context) {/templ Index(c echo.Context, meta layout.PageMeta) {/' views/about/index.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/about/index.templ
fi

# Update events/index.templ
if grep -q "templ Index(c echo.Context) {" views/events/index.templ; then
    sed -i '' 's/templ Index(c echo.Context) {/templ Index(c echo.Context, meta layout.PageMeta) {/' views/events/index.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/events/index.templ
fi

# Update contact/index.templ
if grep -q "templ Index(c echo.Context) {" views/contact/index.templ; then
    sed -i '' 's/templ Index(c echo.Context) {/templ Index(c echo.Context, meta layout.PageMeta) {/' views/contact/index.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/contact/index.templ
fi

# Update portfolio/index.templ
if grep -q "templ Index(c echo.Context) {" views/portfolio/index.templ; then
    sed -i '' 's/templ Index(c echo.Context) {/templ Index(c echo.Context, meta layout.PageMeta) {/' views/portfolio/index.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/portfolio/index.templ
fi

# Update innovation/index.templ
if grep -q "templ Index(c echo.Context) {" views/innovation/index.templ; then
    sed -i '' 's/templ Index(c echo.Context) {/templ Index(c echo.Context, meta layout.PageMeta) {/' views/innovation/index.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/innovation/index.templ
fi

# Update innovation/manufacturing.templ
if grep -q "templ Manufacturing(c echo.Context) {" views/innovation/manufacturing.templ; then
    sed -i '' 's/templ Manufacturing(c echo.Context) {/templ Manufacturing(c echo.Context, meta layout.PageMeta) {/' views/innovation/manufacturing.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/innovation/manufacturing.templ
fi

# Update legal/privacy.templ
if grep -q "templ Privacy(c echo.Context) {" views/legal/privacy.templ; then
    sed -i '' 's/templ Privacy(c echo.Context) {/templ Privacy(c echo.Context, meta layout.PageMeta) {/' views/legal/privacy.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/legal/privacy.templ
fi

# Update legal/terms.templ
if grep -q "templ Terms(c echo.Context) {" views/legal/terms.templ; then
    sed -i '' 's/templ Terms(c echo.Context) {/templ Terms(c echo.Context, meta layout.PageMeta) {/' views/legal/terms.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/legal/terms.templ
fi

# Update legal/shipping.templ
if grep -q "templ Shipping(c echo.Context) {" views/legal/shipping.templ; then
    sed -i '' 's/templ Shipping(c echo.Context) {/templ Shipping(c echo.Context, meta layout.PageMeta) {/' views/legal/shipping.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/legal/shipping.templ
fi

# Update legal/custom_policy.templ
if grep -q "templ CustomPolicy(c echo.Context) {" views/legal/custom_policy.templ; then
    sed -i '' 's/templ CustomPolicy(c echo.Context) {/templ CustomPolicy(c echo.Context, meta layout.PageMeta) {/' views/legal/custom_policy.templ
    sed -i '' 's/@layout.Base(c, layout.PageMeta{[^}]*})/@layout.Base(c, meta)/' views/legal/custom_policy.templ
fi

echo "Template signatures updated successfully!"
