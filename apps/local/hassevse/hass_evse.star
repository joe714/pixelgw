"""
Applet: HASS EVSE
Summary: Home Assistant EVSE charge session
Description: Shows the current EV charge session from a Home Assistant EVSE.
Author: Joe Sunday
"""

load("http.star", "http")
load("humanize.star", "humanize")
load("render.star", "render")
load("schema.star", "schema")
load("time.star", "time")

CACHE_TTL = 10
DEBUG = False

HA_URL = "ha_url"
HA_TOKEN = "ha_token"
DEBUG_STATE = "debug_state"

ENTITY_VEHICLE_CONNECTED = "entity_vehicle_connected"
ENTITY_CONTACTOR = "entity_contactor"
ENTITY_SESSION_POWER = "entity_session_power"
ENTITY_SESSION_ENERGY = "entity_session_energy"
ENTITY_TOTAL_DAY = "entity_total_day"
ENTITY_TOTAL_WEEK = "entity_total_week"
ENTITY_TOTAL_MONTH = "entity_total_month"
ALT_SECONDS = "alt_seconds"

WHITE = "#FFFFFF"
GRAY = "#777777"
GREEN = "#00FF00"
YELLOW = "#FFFF00"
BLUE = "#5599FF"

STATE_LABEL = {
    "charging": "CHARGING",
    "connected": "CONNECTED",
    "complete": "COMPLETE",
}
STATE_COLOR = {
    "charging": GREEN,
    "connected": YELLOW,
    "complete": BLUE,
}

ISO_LAYOUT = "2006-01-02T15:04:05Z07:00"

def _dur(s):
    return time.parse_duration(s)

def _iso(t):
    return t.format(ISO_LAYOUT)

def _binary(state, last_changed):
    return {"state": state, "last_changed": last_changed, "attributes": {}}

def _sensor(state, unit):
    return {
        "state": state,
        "last_changed": _iso(time.now()),
        "attributes": {"unit_of_measurement": unit},
    }

def dummy_entity(entity_key, config):
    scenario = config.get(DEBUG_STATE, "charging")
    now = time.now()

    if entity_key == ENTITY_VEHICLE_CONNECTED:
        if scenario == "disconnected":
            return _binary("off", _iso(now - _dur("5m")))
        if scenario == "connected":
            return _binary("on", _iso(now - _dur("3m")))
        if scenario == "complete":
            return _binary("on", _iso(now - _dur("65m")))
        return _binary("on", _iso(now - _dur("30m")))

    if entity_key == ENTITY_CONTACTOR:
        if scenario == "charging":
            return _binary("on", _iso(now - _dur("23m")))
        if scenario == "complete":
            return _binary("off", _iso(now - _dur("10m")))
        if scenario == "connected":
            return _binary("off", _iso(now - _dur("120m")))
        return _binary("off", _iso(now - _dur("200m")))

    if entity_key == ENTITY_SESSION_POWER:
        return _sensor("7200" if scenario == "charging" else "0", "W")
    if entity_key == ENTITY_SESSION_ENERGY:
        return _sensor("3.5", "kWh")
    if entity_key == ENTITY_TOTAL_DAY:
        return _sensor("12.4", "kWh")
    if entity_key == ENTITY_TOTAL_WEEK:
        return _sensor("58.1", "kWh")
    if entity_key == ENTITY_TOTAL_MONTH:
        return _sensor("203.7", "kWh")
    return None

def fetch_entity(entity_key, config):
    ha_url = config.get(HA_URL)
    if DEBUG or not ha_url:
        return dummy_entity(entity_key, config)

    entity_id = config.get(entity_key)
    if not entity_id:
        return None

    # Strip any trailing slash so we don't build "host:8123//api/states/..",
    # which Home Assistant serves as a 404.
    base = ha_url.rstrip("/")

    rep = http.get(
        base + "/api/states/" + entity_id,
        ttl_seconds = CACHE_TTL,
        headers = {"Authorization": "Bearer " + config.get(HA_TOKEN, "")},
    )
    if rep.status_code != 200:
        return None
    return rep.json()

def is_on(entity):
    return bool(entity) and entity.get("state") == "on"

def entity_last_changed(entity):
    if not entity:
        return None
    lc = entity.get("last_changed")
    if not lc:
        return None
    return time.parse_time(lc)

def secs(later, earlier):
    if not later or not earlier:
        return 0
    return int((later - earlier).seconds)

def fmt_duration(total_seconds):
    total_seconds = int(total_seconds)
    if total_seconds < 0:
        total_seconds = 0
    h = total_seconds // 3600
    m = (total_seconds % 3600) // 60
    if h > 0:
        return "%dh%02d" % (h, m)
    return "%dm" % m

def render_value(entity, convert_to_kw = False, dec = None):
    if not entity:
        return ""
    state = entity.get("state")
    if not state or state == "unknown" or state == "unavailable":
        return ""
    value = float(state)
    unit = entity.get("attributes", {}).get("unit_of_measurement", "")
    if unit == "W" and convert_to_kw:
        unit = "kW"
        value = value / 1000.0
    if dec != None:
        value_str = humanize.float("#,###." + "#" * dec, value)
    elif value < 9.95:
        value_str = humanize.float("#,###.#", value)
    else:
        value_str = humanize.float("#,###.", value)
    if unit == "%":
        return value_str + unit
    if unit:
        return value_str + " " + unit
    return value_str

def session_state(vehicle, contactor, now):
    if not is_on(vehicle):
        return (None, 0)
    t_conn = entity_last_changed(vehicle)
    t_ctr = entity_last_changed(contactor)
    if is_on(contactor):
        return ("charging", secs(now, t_ctr))
    if t_conn and t_ctr and secs(t_ctr, t_conn) > 0:
        return ("complete", secs(t_ctr, t_conn))
    return ("connected", secs(now, t_conn))

# Height of the centered title band at the top of every page. Keeping it
# fixed and shared guarantees the session and totals titles share a baseline.
TITLE_BAND_H = 8

def _page(title, title_color, body):
    return render.Box(
        width = 64,
        height = 32,
        padding = 1,
        child = render.Column(
            expanded = True,
            cross_align = "center",
            children = [
                render.Box(
                    height = TITLE_BAND_H,
                    child = render.Text(title, font = "tom-thumb", color = title_color),
                ),
                body,
            ],
        ),
    )

def session_page(state, seconds, power, energy):
    bottom = []
    if state == "charging":
        power_str = render_value(power, convert_to_kw = True, dec = 1)
        if power_str:
            bottom.append(render.Text(power_str, font = "tom-thumb", color = GREEN))
    energy_str = render_value(energy)
    if energy_str:
        bottom.append(render.Text(energy_str, font = "tom-thumb", color = GRAY))

    body = render.Column(
        expanded = True,
        cross_align = "center",
        main_align = "space_between",
        children = [
            render.Text(fmt_duration(seconds), font = "6x13", color = WHITE),
            render.Row(
                expanded = True,
                main_align = "space_between",
                children = bottom,
            ) if bottom else render.Box(width = 1, height = 1),
        ],
    )
    return _page(STATE_LABEL[state], STATE_COLOR[state], body)

def totals_configured(config):
    for k in (ENTITY_TOTAL_DAY, ENTITY_TOTAL_WEEK, ENTITY_TOTAL_MONTH):
        if not config.get(k):
            return False
    return True

def totals_row(label, entity):
    return render.Row(
        expanded = True,
        main_align = "space_between",
        cross_align = "center",
        children = [
            render.Text(label, font = "tom-thumb", color = GRAY),
            render.Text(render_value(entity), font = "tom-thumb", color = WHITE),
        ],
    )

def totals_page(day, week, month):
    body = render.Column(
        expanded = True,
        cross_align = "center",
        main_align = "space_evenly",
        children = [
            totals_row("Day", day),
            totals_row("Week", week),
            totals_row("Month", month),
        ],
    )
    return _page("EV ENERGY", GREEN, body)

def main(config):
    now = time.now()

    vehicle = fetch_entity(ENTITY_VEHICLE_CONNECTED, config)
    contactor = fetch_entity(ENTITY_CONTACTOR, config)

    pages = []

    state, seconds = session_state(vehicle, contactor, now)
    if state != None:
        power = fetch_entity(ENTITY_SESSION_POWER, config)
        energy = fetch_entity(ENTITY_SESSION_ENERGY, config)
        pages.append(session_page(state, seconds, power, energy))

    if totals_configured(config):
        day = fetch_entity(ENTITY_TOTAL_DAY, config)
        week = fetch_entity(ENTITY_TOTAL_WEEK, config)
        month = fetch_entity(ENTITY_TOTAL_MONTH, config)
        pages.append(totals_page(day, week, month))

    if len(pages) == 0:
        return []
    if len(pages) == 1:
        return [render.Root(child = pages[0])]
    return [render.Root(
        show_full_animation = True,
        delay = int(config.get(ALT_SECONDS, "5")) * 1000,
        child = render.Animation(children = pages),
    )]

def get_schema():
    return schema.Schema(
        version = "1",
        fields = [
            schema.Text(id = HA_URL, name = "Home Assistant URL", desc = "Full URL of your Home Assistant instance.", icon = "book"),
            schema.Text(id = HA_TOKEN, name = "Home Assistant Token", desc = "Long-lived access token (Profile > Security).", icon = "key"),
            schema.Text(id = ENTITY_VEHICLE_CONNECTED, name = "Vehicle connected entity", desc = "binary_sensor for plug presence (e.g. binary_sensor.tesla_director_evse_vehicle_connected)", icon = "plug"),
            schema.Text(id = ENTITY_CONTACTOR, name = "Contactor closed entity", desc = "binary_sensor for contactor closed (e.g. binary_sensor.tesla_director_evse_contactor_closed)", icon = "bolt"),
            schema.Text(id = ENTITY_SESSION_POWER, name = "Session power entity", desc = "sensor in W (e.g. sensor.tesla_director_evse_session_power)", icon = "bolt"),
            schema.Text(id = ENTITY_SESSION_ENERGY, name = "Session energy entity", desc = "sensor in kWh (e.g. sensor.tesla_director_evse_energy_meter_session)", icon = "batteryThreeQuarters"),
            schema.Text(id = ENTITY_TOTAL_DAY, name = "Total energy today", desc = "Meter helper in kWh.", icon = "calendarDay"),
            schema.Text(id = ENTITY_TOTAL_WEEK, name = "Total energy this week", desc = "Meter helper in kWh.", icon = "calendarWeek"),
            schema.Text(id = ENTITY_TOTAL_MONTH, name = "Total energy this month", desc = "Meter helper in kWh.", icon = "calendar"),
            schema.Dropdown(
                id = ALT_SECONDS,
                name = "Seconds per page",
                desc = "How long each page shows when both pages are active.",
                icon = "clock",
                default = "5",
                options = [
                    schema.Option(display = "3 sec", value = "3"),
                    schema.Option(display = "5 sec", value = "5"),
                    schema.Option(display = "8 sec", value = "8"),
                    schema.Option(display = "10 sec", value = "10"),
                ],
            ),
        ],
    )
