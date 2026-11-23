import numpy as np


def _values(data, *, asset=False):
    values = np.asarray(data, dtype=float)
    if values.ndim != 1 or not np.isfinite(values).all():
        raise ValueError("Expected a one-dimensional sequence of finite values.")
    if asset and np.any(values <= 0):
        raise ValueError("Asset values must be positive.")
    if not asset and np.any(values < -1):
        raise ValueError("Returns cannot be below -100%.")
    return values


def total_return(asset):
    """Return the fractional change from initial to final asset value."""
    asset = _values(asset, asset=True)
    return float(asset[-1] / asset[0] - 1) if asset.size >= 2 else float("nan")


def cagr(asset, periods_per_year=365):
    """Compound growth annualized over the intervals between asset values."""
    if not np.isfinite(periods_per_year) or periods_per_year <= 0:
        raise ValueError("periods_per_year must be positive and finite.")
    asset = _values(asset, asset=True)
    if asset.size < 2:
        return float("nan")
    return float((asset[-1] / asset[0]) ** (periods_per_year / (asset.size - 1)) - 1)


def max_drawdown(asset):
    """Return the worst peak-to-trough change as a negative fraction."""
    asset = _values(asset, asset=True)
    if asset.size < 2:
        return float("nan")
    return float(np.min(asset / np.maximum.accumulate(asset) - 1))


def annualized_volatility(returns, periods_per_year=365):
    """Annualized sample standard deviation of periodic simple returns."""
    if not np.isfinite(periods_per_year) or periods_per_year <= 0:
        raise ValueError("periods_per_year must be positive and finite.")
    returns = _values(returns)
    if returns.size < 2:
        return float("nan")
    return float(returns.std(ddof=1) * np.sqrt(periods_per_year))


def sharpe_ratio(returns, periods_per_year=365, risk_free_rate=0.0):
    """Annualized Sharpe; risk_free_rate is an effective annual rate."""
    volatility = annualized_volatility(returns, periods_per_year)
    if not np.isfinite(risk_free_rate) or risk_free_rate <= -1:
        raise ValueError("risk_free_rate must be finite and greater than -1.")
    returns = _values(returns)
    if not np.isfinite(volatility) or volatility == 0:
        return float("nan")
    periodic_rate = (1 + risk_free_rate) ** (1 / periods_per_year) - 1
    return float((returns.mean() - periodic_rate) * periods_per_year / volatility)


def sortino_ratio(returns, periods_per_year=365, target_return=0.0):
    """Annualized Sortino; target_return is an effective annual rate.

    Downside deviation uses squared shortfalls averaged across all periods.
    """
    if not np.isfinite(periods_per_year) or periods_per_year <= 0:
        raise ValueError("periods_per_year must be positive and finite.")
    if not np.isfinite(target_return) or target_return <= -1:
        raise ValueError("target_return must be finite and greater than -1.")
    returns = _values(returns)
    if returns.size < 2:
        return float("nan")
    excess = returns - ((1 + target_return) ** (1 / periods_per_year) - 1)
    downside = np.sqrt(np.mean(np.minimum(excess, 0) ** 2))
    if downside == 0:
        return float("nan")
    return float(excess.mean() / downside * np.sqrt(periods_per_year))


def performance_summary(asset, periods_per_year=365, risk_free_rate=0.0, target_return=0.0):
    """Return all six metrics, deriving simple returns once from asset values."""
    asset = _values(asset, asset=True)
    returns = asset[1:] / asset[:-1] - 1
    return {
        "total_return": total_return(asset),
        "cagr": cagr(asset, periods_per_year),
        "max_drawdown": max_drawdown(asset),
        "annualized_volatility": annualized_volatility(returns, periods_per_year),
        "sharpe_ratio": sharpe_ratio(returns, periods_per_year, risk_free_rate),
        "sortino_ratio": sortino_ratio(returns, periods_per_year, target_return),
    }
