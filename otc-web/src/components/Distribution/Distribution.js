/* eslint-disable no-alert */

import React from 'react';
import PropTypes from 'prop-types';
import styled from 'styled-components';
import moment from 'moment';
import Helmet from 'react-helmet';
import { Flex, Box } from 'grid-styled';
import { FormattedMessage, FormattedHTMLMessage, injectIntl } from 'react-intl';
import { rem } from 'polished';
import { COLORS, SPACE, BOX_SHADOWS, BORDER_RADIUS } from 'config';
import QRCode from 'qrcode.react';

import Button from 'components/Button';
import Container from 'components/Container';
import Footer from 'components/Footer';
import Header from 'components/Header';
import Heading from 'components/Heading';
import Input from 'components/Input';
import Modal, { styles } from 'components/Modal';
import Text from 'components/Text';
import media from '../../utils/media';

import { checkStatus, getAddress, getConfig, checkExchangeStatus } from '../../utils/distributionAPI';

const Wrapper = styled.div`
  background-color: ${COLORS.gray[1]};
  padding: ${rem(SPACE[5])} 0;

  ${media.md.css`
    padding: ${rem(SPACE[7])} 0;
  `}
`;

const Address = Heading.extend`
  display: flex;
  justify-content: space-between;
  word-break: break-all;
  background-color: ${COLORS.gray[0]};
  border-radius: ${BORDER_RADIUS.base};
  box-shadow: ${BOX_SHADOWS.base};
  padding: 1rem;
`;

const StatusModal = ({ status, statusIsOpen, closeModals, skyAddress, intl }) => (
  <Modal
    contentLabel="Status"
    style={styles}
    isOpen={statusIsOpen}
    onRequestClose={closeModals}
  >
    <Heading heavy color="black" fontSize={[2, 3]} my={[3, 5]}>
      <FormattedMessage
        id="distribution.statusFor"
        values={{
          skyAddress,
        }}
      />
    </Heading>

    <Text as="div" color="black" fontSize={[2, 3]} my={[3, 5]}>
      {status.map((status, i) => (
        <p key={i}>
          <FormattedMessage
            id={`distribution.statuses.${status.status}`}
            values={{
              updated: moment.unix(status.updated_at).locale(intl.locale).format('LL LTS'),
            }}
          />
        </p>
      ))}
    </Text>
  </Modal>
);

const StatusErrorMessage = ({ disabledReason }) => (<Flex column>
  <Heading heavy as="h2" fontSize={[5, 6]} color="black" mb={[4, 6]}>
    {(disabledReason === 'coinsSoldOut') ?
      <FormattedMessage id="distribution.errors.coinsSoldOut" /> :
      <FormattedMessage id="distribution.headingEnded" />}
  </Heading>
  <Text heavy color="black" fontSize={[2, 3]} as="div">
    <FormattedHTMLMessage id="distribution.ended" />
  </Text>
</Flex>);

const DistributionFormInfo = ({ sky_btc_exchange_rate, balance }) => (
  <div>
    <Heading heavy as="h2" fontSize={[5, 6]} color="black" mb={[4, 6]}>
      <FormattedMessage id="distribution.heading" />
    </Heading>
    <Text heavy color="black" fontSize={[2, 3]} mb={[4, 6]} as="div">
      <FormattedMessage
        id="distribution.rate"
        values={{
          rate: +sky_btc_exchange_rate,
        }}
      />
    </Text>
    <Text heavy color="black" fontSize={[2, 3]} mb={[4, 6]} as="div">
      <FormattedMessage
        id="distribution.inventory"
        values={{
          coins: balance && balance.coins,
        }}
      />
    </Text>

    <Text heavy color="black" fontSize={[2, 3]} as="div">
      <FormattedHTMLMessage id="distribution.instructions" />
    </Text>
  </div>);

const DistributionForm = ({
  sky_btc_exchange_rate,
  balance,
  intl,

  address,
  handleChange,

  drop_address,
  getAddress,
  addressLoading,

  checkStatus,
  statusLoading,
}) => (
    <Flex justify="center">
      <Box width={[1 / 1, 1 / 1, 2 / 3]} py={[5, 7]}>
        <DistributionFormInfo sky_btc_exchange_rate={sky_btc_exchange_rate} balance={balance} />

        <Input
          placeholder={intl.formatMessage({ id: 'distribution.enterAddress' })}
          value={address}
          onChange={handleChange}
        />

        {drop_address && <Address heavy color="black" fontSize={[2, 3]} as="div">
          <Box>
            <strong><FormattedHTMLMessage id="distribution.btcAddress" />: </strong>
            {drop_address}
          </Box>
          <Box px={5}>
            <QRCode value={drop_address} size={64} />
          </Box>
        </Address>}

        <div>
          <Button
            big
            onClick={getAddress}
            color="white"
            bg="base"
            mr={[2, 5]}
            fontSize={[1, 3]}
          >
            {addressLoading
              ? <FormattedMessage id="distribution.loading" />
              : <FormattedMessage id="distribution.getAddress" />}
          </Button>

          <Button
            onClick={checkStatus}
            color="base"
            big
            outlined
            fontSize={[1, 3]}
          >
            {statusLoading
              ? <FormattedMessage id="distribution.loading" />
              : <FormattedMessage id="distribution.checkStatus" />}
          </Button>
        </div>
      </Box>
    </Flex>);

class Distribution extends React.Component {
  state = {
    status: [],
    skyAddress: null,
    drop_address: '',
    statusIsOpen: false,
    addressLoading: false,
    statusLoading: false,
    enabled: true,
    // TODO: These values should be taken from the OTC API
    sky_btc_exchange_rate: 1,
    balance: { coins: 2 }
  };

  checkExchangeStatus = () => {
    return checkExchangeStatus()
      .then((status) => {
        if (status.error !== '') {
          this.setState({
            disabledReason: 'coinsSoldOut',
            balance: status.balance,
            enabled: false,
          });
        } else {
          this.setState({
            balance: status.balance,
          });
        }
      });
  }

  getConfig = () => {
    return getConfig().then(config => this.setState({ ...config }));
  }

  getAddress = () => {
    if (!this.state.skyAddress) {
      return alert(
        this.props.intl.formatMessage({
          id: 'distribution.errors.noSkyAddress',
        }),
      );
    }

    this.setState({
      addressLoading: true,
    });

    return getAddress(this.state.skyAddress)
      .then((res) => {
        this.setState({
          drop_address: res.drop_address,
        });
      })
      .catch((err) => {
        alert(err.message);
      })
      .then(() => {
        this.setState({
          addressLoading: false,
        });
      });
  }

  handleChange = (event) => {
    this.setState({
      skyAddress: event.target.value,
    });
  }

  closeModals = () => {
    this.setState({
      statusIsOpen: false,
    });
  }

  checkStatus = () => {
    if (!this.state.drop_address) {
      return alert(
        this.props.intl.formatMessage({
          id: 'distribution.errors.noDropAddress',
        }),
      );
    }

    this.setState({
      statusLoading: true,
    });

    return checkStatus({ drop_address: this.state.drop_address, drop_currency: 'BTC' })
      .then((res) => {
        this.setState({
          statusIsOpen: true,
          status: res,
          statusLoading: false,
        });
      })
      .catch((err) => {
        alert(err.message);
      });
  }

  render = () => {
    const { intl } = this.props;
    const {
      statusIsOpen,
      skyAddress,
      status,
      disabledReason,
      enabled,
      sky_btc_exchange_rate,
      balance,
      address,
      drop_address,
      addressLoading,
      statusLoading } = this.state;
    return (
      <div>
        <Helmet>
          <title>{intl.formatMessage({ id: 'distribution.title' })}</title>
        </Helmet>

        <Header external />

        <Wrapper>
          <StatusModal
            statusIsOpen={statusIsOpen}
            closeModals={this.closeModals}
            skyAddress={skyAddress}
            intl={intl}
            status={status}
          />

          <Container>
            {!enabled
              ? <StatusErrorMessage disabledReason={disabledReason} />
              : <DistributionForm
                sky_btc_exchange_rate={sky_btc_exchange_rate}
                balance={balance}
                intl={intl}

                address={address}
                handleChange={this.handleChange}

                drop_address={drop_address}
                getAddress={this.getAddress}
                addressLoading={addressLoading}

                checkStatus={this.checkStatus}
                statusLoading={statusLoading}
              />}
          </Container>
        </Wrapper>

        <Footer external />
      </div>
    );
  }
}

Distribution.propTypes = {
  intl: PropTypes.shape({
    formatMessage: PropTypes.func.isRequired,
  }).isRequired,
};

export default injectIntl(Distribution);
