const mermaidClassDefs =
    `classDef default font-size:16pt\n`
    + `classDef tabsiz_1 fill:#FFFF66\n`
    + `classDef tabsiz_2 fill:#FFD744\n`
    + `classDef tabsiz_3 fill:#FFAF22\n`
    + `classDef tabsiz_4 fill:#96B6F5\n`
    + `classDef tabsiz_5 fill:#70A0F0\n`
    + `classDef tabsiz_6 fill:#4A8AEC\n`
    + `classDef tabsiz_7 fill:#769A62\n`
    + `classDef tabsiz_8 fill:#60804E\n`
    + `classDef tabsiz_9 fill:#4B663B\n`
    + `classDef tabsiz_10 fill:#CB4040\n`;

// globals, used for filtering
var allTables = [];
var allFks = [];
var panZoomInstance = null;

window.addEventListener('resize', function () {
    panZoomInstance?.resize();
    panZoomInstance?.fit();
    panZoomInstance?.center();
});

// UI elements
const connectBtn = document.getElementById('connectBtn');
const disconnectBtn = document.getElementById('disconnectBtn');
const connectInfo = document.getElementById('connectInfo');
const info = document.getElementById('info');
const searchInput = document.getElementById('search');
const schemaSelect = document.getElementById('schemaFilter');
const popup = document.getElementById('popup');
const filterText = document.getElementById('filterText');
const detailsDialog = document.getElementById('detailsDialog');
const detailsTableName = document.getElementById('detailsTableName');
const detailsTableDesc = document.getElementById('detailsTableDesc');

function showPopup(payload) {
    popup.innerText = payload;
    popup.style.left = (event.pageX + 10) + 'px';
    popup.style.top = (event.pageY + 10) + 'px';
    popup.style.display = 'block';
}

function hidePopup() {
    popup.style.display = 'none';
}

function handleEntityClick(entityName, details) {
    //alert(`Clicked on table: ${entityName}\n\n${details}`);
    //detailsContent.innerHTML = `<h3>Table: ${entityName}</h3>\n\n<pre>${details}</pre>`;
    detailsTableName.textContent = entityName
    detailsTableDesc.innerHTML = details

    detailsDialog.style.width = ""
    detailsDialog.style.height = ""
    detailsDialog.dispatchEvent(new Event('resize'))

    detailsDialog.showModal();
}

function closeDialog() {
    detailsDialog.close();
}

async function getConnect() {
    const res = await fetch('/api/getConnect');
    if (res.ok) {
        const body = await res.json();
        if (body?.config.type) {
            document.getElementById('dbType').value = body?.config.type || 'postgres';
            document.getElementById('dsn').value = body?.config.dsn || '';
            document.getElementById('host').value = body?.config.host || '';
            document.getElementById('port').value = body?.config.port || '';
            document.getElementById('username').value = body?.config.username || '';
            document.getElementById('password').value = body?.config.password || '';
            document.getElementById('database_name').value = body?.config.database_name || '';
        }
    }
}

async function postConnect(payload) {
    connectInfo.innerText = 'Connecting...';
    connectBtn.disabled = true;
    try {
        const res = await fetch('/api/connect', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (!res.ok) {
            const txt = await res.text();
            connectInfo.innerText = 'Connection failed: ' + txt;
            connectBtn.disabled = false;
            return null;
        }
        const body = await res.json();
        connectInfo.innerText = 'Connected. Tables: ' + (body.schema.tables?.length || 0);
        return body.schema;
    } catch (err) {
        connectInfo.innerText = 'Connection error: ' + err.message;
        return null;
    } finally {
        connectBtn.disabled = false;
    }
}

function buildPayloadFromForm() {
    const type = document.getElementById('dbType').value;
    const dsn = document.getElementById('dsn').value.trim();
    if (dsn) {
        return { type, dsn };
    }
    const host = document.getElementById('host').value;
    const port = parseInt(document.getElementById('port').value) || undefined;
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const database_name = document.getElementById('database_name').value;
    return { type, host, port, username, password, database_name };
}

connectBtn.addEventListener('click', async () => {
    const payload = buildPayloadFromForm();
    const schema = await postConnect(payload);
    if (schema) renderSchema(schema);
});

disconnectBtn.addEventListener('click', () => {
    connectInfo.innerText = 'Disconnected (refresh to clear server state)';
});

document.getElementById('reload').addEventListener('click', load);

// filtering helpers
async function applyFiltersAndRender() {
    const q = searchInput.value.trim().toLowerCase();
    const schemaSel = schemaSelect.value.toLowerCase(); // empty == all
    filterText.innerText = `Filters - Schema: ${schemaSel || 'All'}, Search: ${q || 'None'}`;

    const filteredTables = allTables.filter(t => {
        const schema = t.schema?.toLowerCase() || '';
        const tabnam = ((schema ? schema + '.' : '') + t.name).toLowerCase();
        const matchesSchema = !schemaSel || schema === schemaSel;
        const matchesQuery = !q || tabnam.includes(q);
        return matchesSchema && matchesQuery;
    });
    //alert(`Filtered tables count: ${filteredTables.length}`); // for debugging
    //alert(JSON.stringify(filteredTables, null, 2)); // for debugging

    const filteredFks = allFks.filter(fk => {
        const fromSchema = fk.from_schema?.toLowerCase() || '';
        const fromTab = ((fromSchema ? fromSchema + '.' : '') + fk.from_table).toLowerCase();
        const toSchema = fk.to_schema?.toLowerCase() || '';
        const toTab = ((toSchema ? toSchema + '.' : '') + fk.to_table).toLowerCase();
        const matchesSchema = !schemaSel || fromSchema === schemaSel || toSchema === schemaSel;
        const matchesQuery = !q || fromTab.includes(q) || toTab.includes(q);
        return matchesSchema && matchesQuery;
    });
    //alert(`Filtered FKs count: ${filteredFks.length}`); // for debugging
    //alert(JSON.stringify(filteredFks, null, 2)); // for debugging

    const [mermaidCode, entityDetails] = jsonToMermaidERD(filteredTables, filteredFks);
    //await navigator.clipboard.writeText( mermaidCode );  // copy mermaid code to clipboard
    //alert( mermaidCode );  // for debugging
    //alert( entityDetails );  // for debugging

    const { svg } = await mermaid.render('mySvgId', mermaidCode);
    insertSvgEvents(svg, document.getElementById('mermaidContainer'), '#mySvgId', entityDetails);
}

searchInput.addEventListener('input', applyFiltersAndRender);
schemaSelect.addEventListener('change', applyFiltersAndRender);

function getDetailsForTable(tableName) {
    const table = allTables.find(t => ((t.schema ? t.schema + '.' : '') + t.name) === tableName);
    if (!table) return 'No details found for table: ' + tableName;

    let details = '';
    if (table.size8kPages) {
        details += `<p><span class="detailLabel">Size:</span> approx. ${(table.size8kPages / 128).toFixed(2)} MB (${table.size8kPages.toLocaleString()} x 8k pages)</p>`;
    }

    if (table.comment) {
        details += `<p><span class="detailLabel">Comment:</span> ${table.comment}</p>`;
    }

    let outboundForeignKeys = '';
    let inboundForeignKeys = '';
    allFks.forEach(fk => {
        const fromTab = (fk.from_schema ? fk.from_schema + '.' : '') + fk.from_table;
        const toTab = (fk.to_schema ? fk.to_schema + '.' : '') + fk.to_table;
        if (fromTab === tableName) {
            outboundForeignKeys += `<tr><td>${fk.constraint || 'FK'}</td><td>${fk.from_column}</td><td>${toTab}</td><td>${fk.to_column}</td></tr>`
        } else if (toTab === tableName) {
            inboundForeignKeys += `<tr><td>${fromTab}</td><td>${fk.constraint}</td><td>${fk.from_column}</td><td>${fk.to_column}</td></tr>`
        }
    })
    if (outboundForeignKeys) {
        details += `<table><caption>Foreign Keys:</caption><thead><tr><th>Constraint Name</th><th>Columns</th><th>Target Table</th><th>Target Columns</th></tr></thead><tbody>${outboundForeignKeys}</tbody></table>`
    }
    if (inboundForeignKeys) {
        details += `<table><caption>Table is referenced by:</caption><thead><tr><th>Referencing Table</th><th>Referencing Constraint</th><th>Referencing Columns</th><th>Columns</th></tr></thead><tbody>${inboundForeignKeys}</tbody>`
    }

    return details;
}

function cleanColumnType(columnType) {
    // replace whitespaces with unicode en space (U+2002)
    // replace all non alphanumeric characters (except spaces, already handled) with empty string
    return columnType.replace(/\s+/g, '\u{2002}').replace(/[^a-zA-Z0-9\s]/g, '').trim();
}

function jsonToMermaidERD(tables, fks) {
    let mermaidSyntax = 'erDiagram\ndirection BT\n\n';
    let entityDetails = {};

    // 1. Add Tables and Columns
    tables?.forEach(table => {
        const tabnam = (table.schema ? table.schema + '.' : '') + table.name
        const tabsiz = table.size8kPages ? Math.min(Math.trunc(Math.log10(table.size8kPages)), 9) + 1 : 1;
        mermaidSyntax += `  "${tabnam}":::tabsiz_${tabsiz} {\n`;
        table.columns.forEach(column => {
            // Add a key indicator if specified in JSON
            const keyIndicator = column.pk ? 'PK' : '';
            mermaidSyntax += `    ${cleanColumnType(column.type)} ${column.name} ${keyIndicator}\n`;
        });
        mermaidSyntax += `  }\n`;
        entityDetails[`${tabnam}`] = getDetailsForTable(tabnam);
    });

    // 2. Add Foreign Keys
    fks?.forEach(fk => {
        const fromTab = (fk.from_schema ? fk.from_schema + '.' : '') + fk.from_table
        const toTab = (fk.to_schema ? fk.to_schema + '.' : '') + fk.to_table
        const constraint = (fk.constraint ? fk.constraint : 'FK');
        mermaidSyntax += `  "${fromTab}" }|--|| "${toTab}" : "${constraint}"\n`;
        // if fromTab or toTab not already in entityDetails, add them
        if (!(fromTab in entityDetails)) {
            entityDetails[`${fromTab}`] = getDetailsForTable(fromTab);
        }
        if (!(toTab in entityDetails)) {
            entityDetails[`${toTab}`] = getDetailsForTable(toTab);
        }
    });

    // 3. Add classDefinitions for table sizes
    mermaidSyntax += mermaidClassDefs;

    return [mermaidSyntax, entityDetails];
}

// The render function and the manual binding logic
const insertSvgEvents = function (svgCode, element, svgId, entityDetails) {
    // clear previous content before loading new SVG
    document.getElementById(svgId)?.remove(); // remove previous instance
    element.innerHTML = '';
    element.innerHTML = svgCode;

    if (svgId) {
        panZoomInstance = svgPanZoom(svgId, {
            zoomEnabled: true,
            panEnabled: true,
            controlIconsEnabled: true,
            fit: true,
            center: true
        });
        panZoomInstance.resize();
        panZoomInstance.fit();
        panZoomInstance.center();
    }

    // Manually find and attach click listeners to the SVG elements
    // The IDs are typically structured as 'entity-ENTITYNAME-...' in mermaid SVGs
    for (const entityName in entityDetails) {
        // Find the specific group element for the entity in the SVG
        // use querySelectorAll to handle multiple groups for the same entity caused by circular references in the diagram
        const entityGroups = element.querySelectorAll(`[id^="entity-${entityName}-"]`);
        for (const entityGroup of entityGroups) {
            entityGroup.addEventListener('click', function () {
                // Pass the entity name to the global handler
                window.handleEntityClick(entityName, entityDetails[entityName]);
            });
            entityGroup.addEventListener('mouseenter', showPopup.bind(null, `Table: ${entityName}`));
            entityGroup.addEventListener('mouseleave', hidePopup);
            // Optional: add a cursor style to indicate clickability
            entityGroup.style.cursor = 'pointer';
        }
    }
};

async function renderSchema(s) {
    document.getElementById('mermaidContainer').innerHTML = 'Rendering...';

    allTables = s?.tables || [];
    allFks = s?.foreign_keys || [];
    info.innerText = `Tables: ${allTables.length}, FKs: ${allFks.length}`;

    // build schema selector
    const schemaSet = new Set();
    allTables.forEach(t => { schemaSet.add((t.schema || '').toString() || ''); });

    // clear and repopulate
    schemaSelect.innerHTML = '<option value="">All schemas</option>';
    Array.from(schemaSet).sort().forEach(schemaName => {
        if (schemaName === '') return;
        const opt = document.createElement('option');
        opt.value = schemaName;
        opt.innerText = schemaName;
        schemaSelect.appendChild(opt);
    });

    // apply any active filters (search or schema)
    applyFiltersAndRender();
}

async function showLegend() {
    const mermaidCode =
        `erDiagram\n`
        + `    "<10":::tabsiz_1 {}\n`
        + `    "<100":::tabsiz_2 {}\n`
        + `    "<1k":::tabsiz_3 {}\n`
        + `    "<10k":::tabsiz_4 {}\n`
        + `    "<100k":::tabsiz_5 {}\n`
        + `    "<1M":::tabsiz_6 {}\n`
        + `    "<10M":::tabsiz_7 {}\n`
        + `    "<100M":::tabsiz_8 {}\n`
        + `    "<1G":::tabsiz_9 {}\n`
        + `    ">1G":::tabsiz_10 {}\n\n`
        + mermaidClassDefs;

    const { svg } = await mermaid.render('myLegendId', mermaidCode);
    insertSvgEvents(svg, document.getElementById('mermaidLegend'), null, null);
}

async function load() {
    info.innerText = 'Loading...';
    showLegend();
    getConnect();
    try {
        const res = await fetch('/api/schema');
        if (!res.ok) {
            const txt = await res.text();
            info.innerText = 'No active connection: ' + txt;
            return;
        }
        const s = await res.json();
        if (!s) {
            info.innerText = 'Load error: empty schema';
            return;
        }
        renderSchema(s);
    } catch (err) {
        info.innerText = 'Load error: ' + err.message;
    }
}

// On db type change, adjust defaults (helpful UX)
document.getElementById('dbType').addEventListener('change', (e) => {
    const t = e.target.value;
    if (t === 'mysql') { document.getElementById('port').value = 3306; }
    else if (t === 'sqlserver') { document.getElementById('port').value = 1433; }
    else if (t === 'godror') { document.getElementById('port').value = 1521; }
    else if (t === 'sqlite') {
        document.getElementById('host').value = '';
        document.getElementById('port').value = '';
        document.getElementById('username').value = '';
        document.getElementById('password').value = '';
    } else { document.getElementById('port').value = 5432; }
});

// initial load (if server has an active connection)
load();

