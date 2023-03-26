/* jshint node: true */

module.exports = function(environment) {
  var ENV = {
    modulePrefix: 'open-social-pool',
    environment: environment,
    rootURL: '/kaspa/',
    locationType: 'hash',
    EmberENV: {
      FEATURES: {
        // Here you can enable experimental features on an ember canary build
        // e.g. 'with-controller': true
      }
    },

    APP: {
      // API host and port
      ApiUrl: '/kaspa/',
      PoolName: 'KASPA SOLO !!! BETA !!!',
      CompanyName: 'Mine to buy a cool car :-)',

      // HTTP mining endpoint
      HttpHost: 'http://pool.lamba.top',
      HttpPort: 18830,

      // Stratum mining endpoint
      StratumHost: 'pool.lamba.top',
      StratumPort: 18030,

      // Fee and payout details
      PoolFee: '0.5%',
      PayoutThreshold: '1.0',
      PayoutInterval: '30m',

      // For network hashrate (change for your favourite fork)
      BlockTime: 1,
      BlockReward: 1,
      Unit: 'KASPA',

    }
  };

  if (environment === 'development') {
  }

  if (environment === 'test') {
  }

  if (environment === 'production') {
  }

  return ENV;
};
